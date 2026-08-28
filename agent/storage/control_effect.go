package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type PendingControlEffect struct {
	CommandID  string
	Revision   int64
	Kind       string
	ReceivedAt time.Time
}

func validControlKind(kind string) bool {
	switch kind {
	case "pause_monitoring", "resume_monitoring", "block_now", "clear_manual_block":
		return true
	default:
		return false
	}
}

// StageControlEffect durably retains the latest control command until the
// daemon confirms its effect on the graphical session.
func (s *Store) StageControlEffect(ctx context.Context, commandID string, revision int64, kind string, now time.Time) error {
	if commandID == "" || revision <= 0 || !validControlKind(kind) || now.IsZero() {
		return errors.New("invalid pending control effect")
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO pending_control_effect(singleton_id, command_id, revision, kind, received_at)
		VALUES (1, ?, ?, ?, ?)
		ON CONFLICT(singleton_id) DO UPDATE SET
			command_id=excluded.command_id,
			revision=excluded.revision,
			kind=excluded.kind,
			received_at=excluded.received_at`, commandID, revision, kind, formatTime(now))
	if err != nil {
		return fmt.Errorf("stage control effect: %w", err)
	}
	return nil
}

func (s *Store) PendingControlEffect(ctx context.Context) (PendingControlEffect, bool, error) {
	var effect PendingControlEffect
	var receivedAt string
	err := s.db.QueryRowContext(ctx, `
		SELECT command_id, revision, kind, received_at
		FROM pending_control_effect WHERE singleton_id=1`,
	).Scan(&effect.CommandID, &effect.Revision, &effect.Kind, &receivedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return PendingControlEffect{}, false, nil
	}
	if err != nil {
		return PendingControlEffect{}, false, err
	}
	effect.ReceivedAt, err = parseTime(receivedAt)
	if err != nil {
		return PendingControlEffect{}, false, err
	}
	return effect, true, nil
}

// CompleteControlEffect makes the command eligible for acknowledgement only
// after the daemon has observed the requested graphical result.
func (s *Store) CompleteControlEffect(ctx context.Context, commandID string, now time.Time) error {
	if commandID == "" || now.IsZero() {
		return errors.New("command id and completion time are required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM pending_control_effect WHERE singleton_id=1 AND command_id=?`, commandID)
	if err != nil {
		return fmt.Errorf("remove completed control effect: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed == 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO applied_command(id, applied_at) VALUES (?, ?)`, commandID, formatTime(now)); err != nil {
		return fmt.Errorf("record completed control effect: %w", err)
	}
	return tx.Commit()
}

func (s *Store) ForgetAppliedCommandIDs(ctx context.Context, commandIDs []string) error {
	if len(commandIDs) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, commandID := range commandIDs {
		if commandID == "" {
			return errors.New("command id is required")
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM applied_command WHERE id=?`, commandID); err != nil {
			return fmt.Errorf("forget acknowledged command: %w", err)
		}
	}
	return tx.Commit()
}
