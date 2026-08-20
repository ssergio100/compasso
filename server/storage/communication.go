package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultCommunicationRetentionDays = 30
	communicationCleanupInterval      = time.Hour
)

type CommunicationLog struct {
	ID         int64             `json:"id"`
	DeviceID   string            `json:"device_id"`
	Source     string            `json:"source"`
	Target     string            `json:"target"`
	Operation  string            `json:"operation"`
	Result     string            `json:"result"`
	HTTPStatus int               `json:"http_status,omitempty"`
	DurationMS int64             `json:"duration_ms,omitempty"`
	Summary    string            `json:"summary"`
	Details    map[string]string `json:"details"`
	CreatedAt  time.Time         `json:"created_at"`
}

func (s *Store) AppendCommunicationLog(ctx context.Context, event CommunicationLog, now time.Time) (CommunicationLog, error) {
	if err := validateCommunicationLog(event, now); err != nil {
		return CommunicationLog{}, err
	}
	details, err := json.Marshal(event.Details)
	if err != nil {
		return CommunicationLog{}, fmt.Errorf("encode communication details: %w", err)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return CommunicationLog{}, err
	}
	defer tx.Rollback()
	if err := cleanupCommunicationLogsIfDue(ctx, tx, now); err != nil {
		return CommunicationLog{}, err
	}
	result, err := tx.ExecContext(ctx, `
		INSERT INTO communication_log(
			device_id, source, target, operation, result, http_status,
			duration_ms, summary, details_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.DeviceID, event.Source, event.Target, event.Operation, event.Result,
		nullablePositiveInt(event.HTTPStatus), nullableDuration(event.DurationMS),
		event.Summary, string(details), formatTime(now),
	)
	if err != nil {
		return CommunicationLog{}, fmt.Errorf("store communication log: %w", err)
	}
	event.ID, err = result.LastInsertId()
	if err != nil {
		return CommunicationLog{}, fmt.Errorf("load communication log id: %w", err)
	}
	event.CreatedAt = now
	if err := tx.Commit(); err != nil {
		return CommunicationLog{}, err
	}
	return event, nil
}

func (s *Store) ListCommunicationLogs(ctx context.Context, deviceID string, afterID int64, limit int) ([]CommunicationLog, error) {
	if deviceID == "" || afterID < 0 || limit < 1 || limit > 500 {
		return nil, errors.New("invalid communication log query")
	}
	if err := ensureDeviceExists(ctx, s.db, deviceID); err != nil {
		return nil, err
	}
	order := "DESC"
	comparison := ""
	arguments := []interface{}{deviceID}
	if afterID > 0 {
		order = "ASC"
		comparison = " AND id>?"
		arguments = append(arguments, afterID)
	}
	arguments = append(arguments, limit)
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, source, target, operation, result, http_status, duration_ms,
		       summary, details_json, created_at
		FROM communication_log WHERE device_id=?`+comparison+`
		ORDER BY id `+order+` LIMIT ?`, arguments...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := make([]CommunicationLog, 0)
	for rows.Next() {
		event := CommunicationLog{DeviceID: deviceID}
		var status, duration sql.NullInt64
		var detailsJSON, created string
		if err := rows.Scan(
			&event.ID, &event.Source, &event.Target, &event.Operation, &event.Result,
			&status, &duration, &event.Summary, &detailsJSON, &created,
		); err != nil {
			return nil, err
		}
		if status.Valid {
			event.HTTPStatus = int(status.Int64)
		}
		if duration.Valid {
			event.DurationMS = duration.Int64
		}
		if err := json.Unmarshal([]byte(detailsJSON), &event.Details); err != nil {
			return nil, fmt.Errorf("decode communication details: %w", err)
		}
		event.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) DeleteCommunicationLogs(ctx context.Context, deviceID string) (int64, error) {
	if deviceID == "" {
		return 0, errors.New("device id is required")
	}
	if err := ensureDeviceExists(ctx, s.db, deviceID); err != nil {
		return 0, err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM communication_log WHERE device_id=?`, deviceID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) CommunicationRetentionDays(ctx context.Context) (int, error) {
	var days int
	err := s.db.QueryRowContext(ctx, `
		SELECT retention_days FROM communication_log_setting WHERE singleton_id=1`,
	).Scan(&days)
	return days, err
}

func (s *Store) SetCommunicationRetentionDays(ctx context.Context, days int, now time.Time) error {
	if days < 1 || days > 365 || now.IsZero() {
		return errors.New("retention must be between 1 and 365 days")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		UPDATE communication_log_setting
		SET retention_days=?, last_cleanup_at=?, updated_at=? WHERE singleton_id=1`,
		days, formatTime(now), formatTime(now)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM communication_log WHERE created_at<?`, formatTime(now.AddDate(0, 0, -days))); err != nil {
		return err
	}
	return tx.Commit()
}

func cleanupCommunicationLogsIfDue(ctx context.Context, tx *sql.Tx, now time.Time) error {
	retentionDays := defaultCommunicationRetentionDays
	var lastCleanup sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT retention_days, last_cleanup_at
		FROM communication_log_setting WHERE singleton_id=1`,
	).Scan(&retentionDays, &lastCleanup); err != nil {
		return err
	}
	if lastCleanup.Valid {
		parsed, err := parseTime(lastCleanup.String)
		if err != nil {
			return err
		}
		if now.Sub(parsed) < communicationCleanupInterval {
			return nil
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM communication_log WHERE created_at<?`, formatTime(now.AddDate(0, 0, -retentionDays))); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE communication_log_setting SET last_cleanup_at=?, updated_at=? WHERE singleton_id=1`,
		formatTime(now), formatTime(now))
	return err
}

func validateCommunicationLog(event CommunicationLog, now time.Time) error {
	if event.DeviceID == "" || now.IsZero() || event.Source == event.Target ||
		!validCommunicationParty(event.Source) || !validCommunicationParty(event.Target) ||
		!validCommunicationResult(event.Result) || event.Operation == "" || len(event.Operation) > 80 ||
		event.Summary == "" || len(event.Summary) > 240 ||
		(event.HTTPStatus != 0 && (event.HTTPStatus < 100 || event.HTTPStatus > 599)) ||
		event.DurationMS < 0 || len(event.Details) > 24 {
		return errors.New("invalid communication log")
	}
	for key, value := range event.Details {
		lowerKey := strings.ToLower(key)
		if key == "" || len(key) > 64 || len(value) > 512 ||
			strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "token") ||
			strings.Contains(lowerKey, "authorization") || strings.Contains(lowerKey, "cookie") ||
			strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "payload") ||
			strings.Contains(lowerKey, "body") {
			return errors.New("unsafe communication detail")
		}
	}
	return nil
}

func validCommunicationParty(value string) bool {
	return value == "agent" || value == "api" || value == "interface"
}

func validCommunicationResult(value string) bool {
	return value == "success" || value == "warning" || value == "error"
}

func nullablePositiveInt(value int) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

func nullableDuration(value int64) interface{} {
	if value == 0 {
		return nil
	}
	return value
}

func ensureDeviceExists(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}, deviceID string) error {
	var exists int
	err := queryer.QueryRowContext(ctx, `SELECT 1 FROM device WHERE id=?`, deviceID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
