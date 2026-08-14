package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ConfirmedSessionState is the last balance explicitly authorized by the
// server for one graphical session. UsageSeconds is the local daily counter at
// that confirmation, allowing the daemon to subtract only later elapsed use.
type ConfirmedSessionState struct {
	Revision         int64
	SessionID        string
	LocalDate        string
	RemainingSeconds int64
	UsageSeconds     int64
	ConfirmedAt      time.Time
}

// CurrentConfirmedSessionState returns the in-memory anchor. It does not query
// SQLite, so the one-second policy loop never rereads quota or bonus totals.
func (s *Store) CurrentConfirmedSessionState() (ConfirmedSessionState, bool) {
	s.sessionStateMu.RLock()
	defer s.sessionStateMu.RUnlock()
	return s.confirmedSessionState, s.hasConfirmedState
}

// SaveConfirmedSessionState persists a new authoritative anchor. Local bonus
// events created after the corresponding heartbeat request remain pending and
// are added so an in-flight response cannot erase newly granted time.
func (s *Store) SaveConfirmedSessionState(ctx context.Context, state ConfirmedSessionState) error {
	if err := validateConfirmedSessionState(state); err != nil {
		return err
	}
	s.sessionStateMu.Lock()
	defer s.sessionStateMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin confirmed session state: %w", err)
	}
	defer tx.Rollback()
	var pendingLocalBonus int64
	if err := tx.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(b.seconds), 0)
		FROM bonus b JOIN pending_event e ON e.uuid=b.uuid
		WHERE b.local_date=? AND b.origin='local'`, state.LocalDate,
	).Scan(&pendingLocalBonus); err != nil {
		return fmt.Errorf("load pending local bonus: %w", err)
	}
	state.RemainingSeconds += pendingLocalBonus
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO daily_usage(local_date, seconds_used, checkpoint_at) VALUES (?, ?, ?)
		ON CONFLICT(local_date) DO UPDATE SET
			seconds_used=MAX(daily_usage.seconds_used, excluded.seconds_used),
			checkpoint_at=CASE
				WHEN excluded.seconds_used >= daily_usage.seconds_used THEN excluded.checkpoint_at
				ELSE daily_usage.checkpoint_at
			END`, state.LocalDate, state.UsageSeconds, formatTime(state.ConfirmedAt)); err != nil {
		return fmt.Errorf("reconcile confirmed server usage: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO confirmed_session_state(
			singleton_id, revision, session_id, local_date, remaining_seconds,
			usage_seconds, confirmed_at
		) VALUES (1, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(singleton_id) DO UPDATE SET
			revision=excluded.revision,
			session_id=excluded.session_id,
			local_date=excluded.local_date,
			remaining_seconds=excluded.remaining_seconds,
			usage_seconds=excluded.usage_seconds,
			confirmed_at=excluded.confirmed_at`,
		state.Revision, state.SessionID, state.LocalDate, state.RemainingSeconds,
		state.UsageSeconds, formatTime(state.ConfirmedAt),
	); err != nil {
		return fmt.Errorf("store confirmed session state: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit confirmed session state: %w", err)
	}
	s.confirmedSessionState = state
	s.hasConfirmedState = true
	return nil
}

func (s *Store) refreshConfirmedSessionStateLocked(ctx context.Context) error {
	var state ConfirmedSessionState
	var confirmedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT revision, session_id, local_date, remaining_seconds,
		       usage_seconds, confirmed_at
		FROM confirmed_session_state WHERE singleton_id=1`,
	).Scan(&state.Revision, &state.SessionID, &state.LocalDate,
		&state.RemainingSeconds, &state.UsageSeconds, &confirmedAt)
	if errors.Is(err, sql.ErrNoRows) {
		s.confirmedSessionState = ConfirmedSessionState{}
		s.hasConfirmedState = false
		return nil
	}
	if err != nil {
		return fmt.Errorf("load confirmed session state: %w", err)
	}
	state.ConfirmedAt, err = parseTime(confirmedAt)
	if err != nil {
		return err
	}
	if err := validateConfirmedSessionState(state); err != nil {
		return fmt.Errorf("invalid stored confirmed session state: %w", err)
	}
	s.confirmedSessionState = state
	s.hasConfirmedState = true
	return nil
}

func validateConfirmedSessionState(state ConfirmedSessionState) error {
	if state.Revision < 0 || state.SessionID == "" || len(state.SessionID) > 128 ||
		state.RemainingSeconds < 0 || state.UsageSeconds < 0 || state.ConfirmedAt.IsZero() {
		return errors.New("confirmed session state has invalid fields")
	}
	return validateLocalDate(state.LocalDate)
}
