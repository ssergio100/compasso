package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"
)

// StoredRoutine is the persistent representation of a recurring routine.
type StoredRoutine struct {
	ID      string
	Name    string
	Days    [7]bool
	Start   time.Duration
	End     time.Duration
	Enabled bool
}

// PolicySnapshot is a complete remotely managed configuration revision.
type PolicySnapshot struct {
	Revision              int64
	MonitoringPaused      bool
	ManualBlock           bool
	WarningMinutes        int
	LocalPasswordVerifier string
	WeeklyQuota           [7]time.Duration
	Routines              []StoredRoutine
	UpdatedAt             time.Time
}

// ReplacePolicy atomically replaces remotely managed state. Consumption,
// bonuses and pending local events are deliberately outside this transaction.
func (s *Store) ReplacePolicy(ctx context.Context, policy PolicySnapshot) error {
	if err := validatePolicy(policy); err != nil {
		return err
	}
	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	routines := append([]StoredRoutine(nil), policy.Routines...)
	sort.Slice(routines, func(i, j int) bool { return routines[i].ID < routines[j].ID })
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin policy replacement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	currentRevision, exists, err := policyRevision(ctx, tx)
	if err != nil {
		return err
	}
	if exists && policy.Revision < currentRevision {
		return fmt.Errorf("%w: received %d, current %d", ErrStalePolicy, policy.Revision, currentRevision)
	}
	if exists && policy.Revision == currentRevision {
		return nil
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM routine_day`); err != nil {
		return fmt.Errorf("clear routine days: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM routine`); err != nil {
		return fmt.Errorf("clear routines: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM weekly_quota`); err != nil {
		return fmt.Errorf("clear weekly quotas: %w", err)
	}

	for weekday, quota := range policy.WeeklyQuota {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO weekly_quota(weekday, seconds_allowed) VALUES (?, ?)`,
			weekday, int64(quota/time.Second),
		); err != nil {
			return fmt.Errorf("store quota for weekday %d: %w", weekday, err)
		}
	}
	for _, routine := range routines {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO routine(id, name, start_second, end_second, enabled) VALUES (?, ?, ?, ?, ?)`,
			routine.ID, routine.Name, int64(routine.Start/time.Second), int64(routine.End/time.Second), boolInt(routine.Enabled),
		); err != nil {
			return fmt.Errorf("store routine %s: %w", routine.ID, err)
		}
		for weekday, selected := range routine.Days {
			if !selected {
				continue
			}
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO routine_day(routine_id, weekday) VALUES (?, ?)`, routine.ID, weekday,
			); err != nil {
				return fmt.Errorf("store day %d for routine %s: %w", weekday, routine.ID, err)
			}
		}
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO policy_state(
			singleton_id, revision, monitoring_paused, manual_block,
			warning_minutes, local_password_verifier, updated_at
		) VALUES (1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(singleton_id) DO UPDATE SET
			revision = excluded.revision,
			monitoring_paused = excluded.monitoring_paused,
			manual_block = excluded.manual_block,
			warning_minutes = excluded.warning_minutes,
			local_password_verifier = excluded.local_password_verifier,
			updated_at = excluded.updated_at`,
		policy.Revision, boolInt(policy.MonitoringPaused), boolInt(policy.ManualBlock),
		policy.WarningMinutes, policy.LocalPasswordVerifier, formatTime(policy.UpdatedAt),
	); err != nil {
		return fmt.Errorf("store policy state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit policy replacement: %w", err)
	}
	policy.Routines = routines
	s.cachedPolicy = clonePolicy(policy)
	s.hasCachedPolicy = true
	return nil
}

// LoadPolicy reconstructs the last complete, committed policy revision.
func (s *Store) LoadPolicy(ctx context.Context) (PolicySnapshot, error) {
	return s.loadPolicyFromDatabase(ctx)
}

// CurrentPolicy returns the in-memory policy used by the one-second daemon
// loop. It is replaced atomically after a synchronized revision is committed.
func (s *Store) CurrentPolicy() (PolicySnapshot, error) {
	s.policyMu.RLock()
	defer s.policyMu.RUnlock()
	if !s.hasCachedPolicy {
		return PolicySnapshot{}, ErrNoPolicy
	}
	return clonePolicy(s.cachedPolicy), nil
}

func (s *Store) refreshPolicyCache(ctx context.Context) error {
	policy, err := s.loadPolicyFromDatabase(ctx)
	if errors.Is(err, ErrNoPolicy) {
		s.policyMu.Lock()
		s.cachedPolicy = PolicySnapshot{}
		s.hasCachedPolicy = false
		s.policyMu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	s.setCachedPolicy(policy)
	return nil
}

func (s *Store) setCachedPolicy(policy PolicySnapshot) {
	s.policyMu.Lock()
	s.cachedPolicy = clonePolicy(policy)
	s.hasCachedPolicy = true
	s.policyMu.Unlock()
}

func clonePolicy(policy PolicySnapshot) PolicySnapshot {
	policy.Routines = append([]StoredRoutine(nil), policy.Routines...)
	return policy
}

func (s *Store) loadPolicyFromDatabase(ctx context.Context) (PolicySnapshot, error) {
	var policy PolicySnapshot
	var paused, manual int
	var updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT revision, monitoring_paused, manual_block, warning_minutes,
		       COALESCE(local_password_verifier, ''), updated_at
		FROM policy_state WHERE singleton_id = 1`,
	).Scan(&policy.Revision, &paused, &manual, &policy.WarningMinutes, &policy.LocalPasswordVerifier, &updatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PolicySnapshot{}, ErrNoPolicy
	}
	if err != nil {
		return PolicySnapshot{}, fmt.Errorf("load policy state: %w", err)
	}
	policy.MonitoringPaused, policy.ManualBlock = paused != 0, manual != 0
	policy.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return PolicySnapshot{}, err
	}

	quotaRows, err := s.db.QueryContext(ctx, `SELECT weekday, seconds_allowed FROM weekly_quota ORDER BY weekday`)
	if err != nil {
		return PolicySnapshot{}, fmt.Errorf("load weekly quotas: %w", err)
	}
	quotaCount := 0
	for quotaRows.Next() {
		var weekday int
		var seconds int64
		if err := quotaRows.Scan(&weekday, &seconds); err != nil {
			_ = quotaRows.Close()
			return PolicySnapshot{}, fmt.Errorf("scan weekly quota: %w", err)
		}
		policy.WeeklyQuota[weekday] = time.Duration(seconds) * time.Second
		quotaCount++
	}
	if err := quotaRows.Close(); err != nil {
		return PolicySnapshot{}, fmt.Errorf("close weekly quota rows: %w", err)
	}
	if quotaCount != 7 {
		return PolicySnapshot{}, fmt.Errorf("stored policy has %d weekly quotas, want 7", quotaCount)
	}

	routineRows, err := s.db.QueryContext(ctx, `
		SELECT id, name, start_second, end_second, enabled FROM routine ORDER BY id`)
	if err != nil {
		return PolicySnapshot{}, fmt.Errorf("load routines: %w", err)
	}
	for routineRows.Next() {
		var routine StoredRoutine
		var start, end int64
		var enabled int
		if err := routineRows.Scan(&routine.ID, &routine.Name, &start, &end, &enabled); err != nil {
			_ = routineRows.Close()
			return PolicySnapshot{}, fmt.Errorf("scan routine: %w", err)
		}
		routine.Start = time.Duration(start) * time.Second
		routine.End = time.Duration(end) * time.Second
		routine.Enabled = enabled != 0
		policy.Routines = append(policy.Routines, routine)
	}
	if err := routineRows.Close(); err != nil {
		return PolicySnapshot{}, fmt.Errorf("close routine rows: %w", err)
	}

	dayRows, err := s.db.QueryContext(ctx, `SELECT routine_id, weekday FROM routine_day ORDER BY routine_id, weekday`)
	if err != nil {
		return PolicySnapshot{}, fmt.Errorf("load routine days: %w", err)
	}
	routineIndexes := make(map[string]int, len(policy.Routines))
	for i := range policy.Routines {
		routineIndexes[policy.Routines[i].ID] = i
	}
	for dayRows.Next() {
		var routineID string
		var weekday int
		if err := dayRows.Scan(&routineID, &weekday); err != nil {
			_ = dayRows.Close()
			return PolicySnapshot{}, fmt.Errorf("scan routine day: %w", err)
		}
		index, ok := routineIndexes[routineID]
		if !ok {
			_ = dayRows.Close()
			return PolicySnapshot{}, fmt.Errorf("routine day references unknown routine %s", routineID)
		}
		policy.Routines[index].Days[weekday] = true
	}
	if err := dayRows.Close(); err != nil {
		return PolicySnapshot{}, fmt.Errorf("close routine day rows: %w", err)
	}
	return policy, nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

func policyRevision(ctx context.Context, queryer queryRower) (int64, bool, error) {
	var revision int64
	err := queryer.QueryRowContext(ctx,
		`SELECT revision FROM policy_state WHERE singleton_id = 1`,
	).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("load current policy revision: %w", err)
	}
	return revision, true, nil
}

func validatePolicy(policy PolicySnapshot) error {
	if policy.Revision < 0 {
		return errors.New("policy revision cannot be negative")
	}
	if policy.WarningMinutes < 0 {
		return errors.New("warning minutes cannot be negative")
	}
	if policy.UpdatedAt.IsZero() {
		return errors.New("policy updated time must be set")
	}
	for weekday, quota := range policy.WeeklyQuota {
		if quota < 0 || quota%time.Second != 0 {
			return fmt.Errorf("quota for weekday %d must be a non-negative whole number of seconds", weekday)
		}
	}
	ids := make(map[string]struct{}, len(policy.Routines))
	for _, routine := range policy.Routines {
		if routine.ID == "" || routine.Name == "" {
			return errors.New("routine id and name cannot be empty")
		}
		if _, exists := ids[routine.ID]; exists {
			return fmt.Errorf("duplicate routine id %s", routine.ID)
		}
		ids[routine.ID] = struct{}{}
		if routine.Start < 0 || routine.Start >= 24*time.Hour || routine.Start%time.Second != 0 ||
			routine.End < 0 || routine.End >= 24*time.Hour || routine.End%time.Second != 0 {
			return fmt.Errorf("routine %s has invalid start or end", routine.ID)
		}
	}
	return nil
}
