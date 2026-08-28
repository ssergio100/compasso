package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// BindEnrollment associates all durable agent state with one server device.
// Existing state without an identity is trusted only for an already-confirmed
// installation, allowing upgrades to preserve valid offline policy.
func (s *Store) BindEnrollment(ctx context.Context, serverURL, deviceID string, trustUnboundState bool) (bool, error) {
	if serverURL == "" || deviceID == "" {
		return false, errors.New("server URL and device ID are required")
	}

	s.policyMu.Lock()
	defer s.policyMu.Unlock()
	s.sessionStateMu.Lock()
	defer s.sessionStateMu.Unlock()

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin enrollment binding: %w", err)
	}
	defer tx.Rollback()

	var storedServerURL, storedDeviceID string
	err = tx.QueryRowContext(ctx, `
		SELECT server_url, device_id FROM enrollment WHERE singleton_id=1`,
	).Scan(&storedServerURL, &storedDeviceID)
	unbound := errors.Is(err, sql.ErrNoRows)
	if err != nil && !unbound {
		return false, fmt.Errorf("load enrollment binding: %w", err)
	}
	reset := (unbound && !trustUnboundState) || (!unbound && (storedServerURL != serverURL || storedDeviceID != deviceID))
	if reset {
		for _, statement := range []string{
			`DELETE FROM routine_day`,
			`DELETE FROM routine`,
			`DELETE FROM weekly_quota`,
			`DELETE FROM policy_state`,
			`DELETE FROM confirmed_session_state`,
			`DELETE FROM pending_control_effect`,
			`DELETE FROM applied_command`,
			`DELETE FROM pending_event`,
			`DELETE FROM bonus`,
			`DELETE FROM daily_usage`,
		} {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return false, fmt.Errorf("clear previous enrollment state: %w", err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO enrollment(singleton_id, server_url, device_id) VALUES (1, ?, ?)
		ON CONFLICT(singleton_id) DO UPDATE SET
			server_url=excluded.server_url, device_id=excluded.device_id`, serverURL, deviceID); err != nil {
		return false, fmt.Errorf("store enrollment binding: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit enrollment binding: %w", err)
	}
	if reset {
		s.cachedPolicy = PolicySnapshot{}
		s.hasCachedPolicy = false
		s.confirmedSessionState = ConfirmedSessionState{}
		s.hasConfirmedState = false
	}
	return reset, nil
}
