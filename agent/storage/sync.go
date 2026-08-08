package storage

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// AppliedCommandIDs are resent as acknowledgements. Keeping them locally makes
// a response retry harmless even when the previous acknowledgement was lost.
func (s *Store) AppliedCommandIDs(ctx context.Context, limit int) ([]string, error) {
	if limit <= 0 {
		return nil, errors.New("command limit must be positive")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM applied_command ORDER BY applied_at DESC, id LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("load applied commands: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) RecordCommandApplied(ctx context.Context, id string, now time.Time) error {
	if id == "" || now.IsZero() {
		return errors.New("command id and applied time are required")
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO applied_command(id, applied_at) VALUES (?, ?)`, id, formatTime(now))
	if err != nil {
		return fmt.Errorf("record applied command: %w", err)
	}
	return nil
}

// ApplyRemoteBonus atomically applies an at-least-once command and records its
// acknowledgement. Repeating the same command never duplicates the bonus.
func (s *Store) ApplyRemoteBonus(ctx context.Context, commandID string, bonus Bonus, now time.Time) error {
	if commandID == "" || now.IsZero() {
		return errors.New("command id and applied time are required")
	}
	if err := validateBonus(bonus); err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO applied_command(id, applied_at) VALUES (?, ?)`, commandID, formatTime(now))
	if err != nil {
		return fmt.Errorf("record remote bonus command: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed != 0 {
		if _, err := tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO bonus(uuid, local_date, seconds, origin, created_at)
			VALUES (?, ?, ?, ?, ?)`, bonus.UUID, bonus.LocalDate, bonus.Seconds, bonus.Origin, formatTime(bonus.CreatedAt)); err != nil {
			return fmt.Errorf("store remote bonus: %w", err)
		}
	}
	return tx.Commit()
}
