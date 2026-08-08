package storage

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Admin struct {
	ID           string
	Login        string
	PasswordHash string
	Active       bool
}

type Device struct {
	ID                    string
	Name                  string
	LastSeenAt            *time.Time
	PolicyRevision        int64
	AppliedPolicyRevision int64
	CreatedAt             time.Time
	Online                bool
}

type Policy struct {
	Revision              int64
	MonitoringPaused      bool
	ManualBlock           bool
	WarningMinutes        int
	LocalPasswordVerifier string
	WeeklyQuota           [7]int64
	Routines              []Routine
	UpdatedAt             time.Time
}

type Routine struct {
	ID      string
	Name    string
	Days    [7]bool
	Start   int64
	End     int64
	Enabled bool
}

type AuditEvent struct {
	ID        string
	Kind      string
	Origin    string
	Details   string
	CreatedAt time.Time
}

type DailySummary struct {
	UsedSeconds  int64
	BonusSeconds int64
}

func (s *Store) BootstrapAdmin(ctx context.Context, login, passwordHash string, now time.Time) (bool, error) {
	login = strings.TrimSpace(login)
	if login == "" || passwordHash == "" || now.IsZero() {
		return false, errors.New("bootstrap admin requires login, password hash and time")
	}
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM admin_user`).Scan(&count); err != nil {
		return false, fmt.Errorf("count administrators: %w", err)
	}
	if count != 0 {
		return false, nil
	}
	id, err := newID()
	if err != nil {
		return false, err
	}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO admin_user(id, login, password_hash, active, created_at, updated_at)
		VALUES (?, ?, ?, 1, ?, ?)`, id, login, passwordHash, formatTime(now), formatTime(now))
	if err != nil {
		return false, fmt.Errorf("bootstrap administrator: %w", err)
	}
	return true, nil
}

func (s *Store) AdminByLogin(ctx context.Context, login string) (Admin, error) {
	var admin Admin
	var active int
	err := s.db.QueryRowContext(ctx,
		`SELECT id, login, password_hash, active FROM admin_user WHERE login = ?`, strings.TrimSpace(login),
	).Scan(&admin.ID, &admin.Login, &admin.PasswordHash, &active)
	if errors.Is(err, sql.ErrNoRows) {
		return Admin{}, ErrNotFound
	}
	if err != nil {
		return Admin{}, fmt.Errorf("load administrator: %w", err)
	}
	admin.Active = active != 0
	return admin, nil
}

func (s *Store) CreateDevice(ctx context.Context, name string, now time.Time) (Device, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 || now.IsZero() {
		return Device{}, errors.New("device requires a name of at most 80 characters and current time")
	}
	id, err := newID()
	if err != nil {
		return Device{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Device{}, err
	}
	defer tx.Rollback()
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO device(id, name, device_token_hash, policy_revision, created_at, updated_at)
		VALUES (?, ?, '', 1, ?, ?)`, id, name, stamp, stamp); err != nil {
		return Device{}, fmt.Errorf("create device: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO policy(device_id, revision, warning_minutes, updated_at) VALUES (?, 1, 10, ?)`, id, stamp); err != nil {
		return Device{}, fmt.Errorf("create device policy: %w", err)
	}
	for weekday := 0; weekday < 7; weekday++ {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO weekly_quota(device_id, weekday, seconds_allowed) VALUES (?, ?, 0)`, id, weekday); err != nil {
			return Device{}, err
		}
	}
	if err := insertAudit(ctx, tx, id, "device_created", map[string]interface{}{"name": name}, now); err != nil {
		return Device{}, err
	}
	if err := tx.Commit(); err != nil {
		return Device{}, err
	}
	return Device{ID: id, Name: name, PolicyRevision: 1, CreatedAt: now}, nil
}

func (s *Store) ListDevices(ctx context.Context) ([]Device, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, last_seen_at, policy_revision, applied_policy_revision, created_at FROM device ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list devices: %w", err)
	}
	defer rows.Close()
	var devices []Device
	for rows.Next() {
		var device Device
		var lastSeen sql.NullString
		var created string
		if err := rows.Scan(&device.ID, &device.Name, &lastSeen, &device.PolicyRevision, &device.AppliedPolicyRevision, &created); err != nil {
			return nil, err
		}
		device.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			value, err := parseTime(lastSeen.String)
			if err != nil {
				return nil, err
			}
			device.LastSeenAt = &value
		}
		devices = append(devices, device)
	}
	return devices, rows.Err()
}

func (s *Store) LoadDevice(ctx context.Context, id string) (Device, Policy, error) {
	var device Device
	var lastSeen sql.NullString
	var created string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, last_seen_at, policy_revision, applied_policy_revision, created_at FROM device WHERE id=?`, id,
	).Scan(&device.ID, &device.Name, &lastSeen, &device.PolicyRevision, &device.AppliedPolicyRevision, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return Device{}, Policy{}, ErrNotFound
	}
	if err != nil {
		return Device{}, Policy{}, err
	}
	device.CreatedAt, _ = parseTime(created)
	if lastSeen.Valid {
		value, parseErr := parseTime(lastSeen.String)
		if parseErr != nil {
			return Device{}, Policy{}, parseErr
		}
		device.LastSeenAt = &value
	}
	policy, err := s.loadPolicy(ctx, id)
	return device, policy, err
}

func (s *Store) RenameDevice(ctx context.Context, id, name string, now time.Time) error {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 80 {
		return errors.New("device name must contain at most 80 characters")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE device SET name=?, updated_at=? WHERE id=?`, name, formatTime(now), id)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return ErrNotFound
	}
	return s.InsertAudit(ctx, id, "device_renamed", map[string]interface{}{"name": name}, now)
}

func (s *Store) DeleteDevice(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM device WHERE id=?`, id)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SaveQuotas(ctx context.Context, deviceID string, quotas [7]int64, warningMinutes int, now time.Time) error {
	for _, seconds := range quotas {
		if seconds < 0 || seconds > 24*60*60 {
			return errors.New("quota must be between 0 and 24 hours")
		}
	}
	if warningMinutes < 0 || warningMinutes > 120 {
		return errors.New("warning must be between 0 and 120 minutes")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for weekday, seconds := range quotas {
		if _, err := tx.ExecContext(ctx, `UPDATE weekly_quota SET seconds_allowed=? WHERE device_id=? AND weekday=?`, seconds, deviceID, weekday); err != nil {
			return err
		}
	}
	revision, err := bumpPolicy(ctx, tx, deviceID, warningMinutes, now)
	if err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, deviceID, "quotas_updated", map[string]interface{}{
		"revision": revision, "weekly_seconds": quotas, "warning_minutes": warningMinutes,
	}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SaveRoutine(ctx context.Context, deviceID string, routine Routine, now time.Time) (string, error) {
	routine.Name = strings.TrimSpace(routine.Name)
	if routine.Name == "" || len(routine.Name) > 80 || routine.Start < 0 || routine.Start >= 86400 || routine.End < 0 || routine.End >= 86400 {
		return "", errors.New("routine has invalid name or time")
	}
	selected := false
	for _, day := range routine.Days {
		selected = selected || day
	}
	if !selected {
		return "", errors.New("routine requires at least one day")
	}
	isNew := routine.ID == ""
	if isNew {
		var err error
		routine.ID, err = newID()
		if err != nil {
			return "", err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if !isNew {
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM routine WHERE id=? AND device_id=?`, routine.ID, deviceID).Scan(&count); err != nil {
			return "", err
		}
		if count == 0 {
			return "", ErrNotFound
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO routine(id, device_id, name, start_second, end_second, enabled)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET name=excluded.name, start_second=excluded.start_second,
		end_second=excluded.end_second, enabled=excluded.enabled
		WHERE routine.device_id=excluded.device_id`, routine.ID, deviceID, routine.Name, routine.Start, routine.End, boolInt(routine.Enabled)); err != nil {
		return "", err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM routine_day WHERE routine_id=?`, routine.ID); err != nil {
		return "", err
	}
	for weekday, enabled := range routine.Days {
		if enabled {
			if _, err := tx.ExecContext(ctx, `INSERT INTO routine_day(routine_id, weekday) VALUES (?, ?)`, routine.ID, weekday); err != nil {
				return "", err
			}
		}
	}
	revision, err := bumpPolicyKeepWarning(ctx, tx, deviceID, now)
	if err != nil {
		return "", err
	}
	if err := insertAudit(ctx, tx, deviceID, "routine_saved", map[string]interface{}{
		"routine_id": routine.ID, "name": routine.Name, "revision": revision,
	}, now); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return routine.ID, nil
}

func (s *Store) DeleteRoutine(ctx context.Context, deviceID, routineID string, now time.Time) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM routine WHERE id=? AND device_id=?`, routineID, deviceID)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return ErrNotFound
	}
	if _, err := bumpPolicyKeepWarning(ctx, tx, deviceID, now); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, deviceID, "routine_deleted", map[string]interface{}{"routine_id": routineID}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) SetLocalPassword(ctx context.Context, deviceID, verifier string, now time.Time) error {
	if verifier == "" {
		return errors.New("local password verifier cannot be empty")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE policy SET local_password_verifier=?, revision=revision+1, updated_at=? WHERE device_id=?`, verifier, formatTime(now), deviceID)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `UPDATE device SET policy_revision=policy_revision+1, updated_at=? WHERE id=?`, formatTime(now), deviceID); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, deviceID, "local_password_changed", map[string]interface{}{"status": "awaiting_sync"}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) ListAudit(ctx context.Context, deviceID string, limit int) ([]AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT uuid, kind, origin, payload_json, created_at FROM audit_event
		WHERE device_id=? ORDER BY created_at DESC, uuid DESC LIMIT ?`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []AuditEvent
	for rows.Next() {
		var event AuditEvent
		var created string
		if err := rows.Scan(&event.ID, &event.Kind, &event.Origin, &event.Details, &created); err != nil {
			return nil, err
		}
		event.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) LoadDailySummary(ctx context.Context, deviceID, localDate string) (DailySummary, error) {
	var summary DailySummary
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE((SELECT seconds_used FROM daily_usage WHERE device_id=? AND local_date=?), 0),
		       COALESCE((SELECT SUM(seconds) FROM bonus WHERE device_id=? AND local_date=?), 0)`,
		deviceID, localDate, deviceID, localDate,
	).Scan(&summary.UsedSeconds, &summary.BonusSeconds)
	if err != nil {
		return DailySummary{}, fmt.Errorf("load daily summary: %w", err)
	}
	return summary, nil
}

func (s *Store) InsertAudit(ctx context.Context, deviceID, kind string, payload interface{}, now time.Time) error {
	return insertAudit(ctx, s.db, deviceID, kind, payload, now)
}

func (s *Store) loadPolicy(ctx context.Context, deviceID string) (Policy, error) {
	var policy Policy
	var paused, blocked int
	var updatedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT revision, monitoring_paused, manual_block, warning_minutes,
		COALESCE(local_password_verifier, ''), updated_at FROM policy WHERE device_id=?`, deviceID,
	).Scan(&policy.Revision, &paused, &blocked, &policy.WarningMinutes, &policy.LocalPasswordVerifier, &updatedAt)
	if err != nil {
		return Policy{}, err
	}
	policy.MonitoringPaused, policy.ManualBlock = paused != 0, blocked != 0
	policy.UpdatedAt, err = parseTime(updatedAt)
	if err != nil {
		return Policy{}, err
	}
	quotaRows, err := s.db.QueryContext(ctx, `SELECT weekday, seconds_allowed FROM weekly_quota WHERE device_id=?`, deviceID)
	if err != nil {
		return Policy{}, err
	}
	for quotaRows.Next() {
		var weekday int
		var seconds int64
		if err := quotaRows.Scan(&weekday, &seconds); err != nil {
			_ = quotaRows.Close()
			return Policy{}, err
		}
		policy.WeeklyQuota[weekday] = seconds
	}
	_ = quotaRows.Close()
	routineRows, err := s.db.QueryContext(ctx, `SELECT id, name, start_second, end_second, enabled FROM routine WHERE device_id=? ORDER BY name, id`, deviceID)
	if err != nil {
		return Policy{}, err
	}
	for routineRows.Next() {
		var routine Routine
		var enabled int
		if err := routineRows.Scan(&routine.ID, &routine.Name, &routine.Start, &routine.End, &enabled); err != nil {
			_ = routineRows.Close()
			return Policy{}, err
		}
		routine.Enabled = enabled != 0
		policy.Routines = append(policy.Routines, routine)
	}
	_ = routineRows.Close()
	indexes := make(map[string]int)
	for index := range policy.Routines {
		indexes[policy.Routines[index].ID] = index
	}
	dayRows, err := s.db.QueryContext(ctx, `
		SELECT rd.routine_id, rd.weekday FROM routine_day rd JOIN routine r ON r.id=rd.routine_id
		WHERE r.device_id=?`, deviceID)
	if err != nil {
		return Policy{}, err
	}
	for dayRows.Next() {
		var id string
		var weekday int
		if err := dayRows.Scan(&id, &weekday); err != nil {
			_ = dayRows.Close()
			return Policy{}, err
		}
		policy.Routines[indexes[id]].Days[weekday] = true
	}
	_ = dayRows.Close()
	return policy, nil
}

type execer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

func insertAudit(ctx context.Context, executor execer, deviceID, kind string, payload interface{}, now time.Time) error {
	id, err := newID()
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = executor.ExecContext(ctx, `
		INSERT INTO audit_event(uuid, device_id, kind, origin, payload_json, created_at)
		VALUES (?, ?, ?, 'web', ?, ?)`, id, deviceID, kind, string(encoded), formatTime(now))
	return err
}

func bumpPolicy(ctx context.Context, tx *sql.Tx, deviceID string, warningMinutes int, now time.Time) (int64, error) {
	if _, err := tx.ExecContext(ctx, `UPDATE policy SET revision=revision+1, warning_minutes=?, updated_at=? WHERE device_id=?`, warningMinutes, formatTime(now), deviceID); err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE device SET policy_revision=policy_revision+1, updated_at=? WHERE id=?`, formatTime(now), deviceID); err != nil {
		return 0, err
	}
	var revision int64
	err := tx.QueryRowContext(ctx, `SELECT revision FROM policy WHERE device_id=?`, deviceID).Scan(&revision)
	return revision, err
}

func bumpPolicyKeepWarning(ctx context.Context, tx *sql.Tx, deviceID string, now time.Time) (int64, error) {
	var warning int
	if err := tx.QueryRowContext(ctx, `SELECT warning_minutes FROM policy WHERE device_id=?`, deviceID).Scan(&warning); err != nil {
		return 0, err
	}
	return bumpPolicy(ctx, tx, deviceID, warning, now)
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func randomID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(bytes)
	parts := []string{encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32]}
	return strings.Join(parts, "-"), nil
}
