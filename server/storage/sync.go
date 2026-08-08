package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	protocol "github.com/sergio/compasso/protocol/v1"
)

var (
	ErrInvalidDeviceCredentials = errors.New("invalid device credentials")
	ErrRevisionAhead            = errors.New("client policy revision is ahead of server")
)

// IssueDeviceToken replaces a device credential and returns the secret once.
// Only its SHA-256 digest is persisted; the token has 256 bits of entropy.
func (s *Store) IssueDeviceToken(ctx context.Context, deviceID string, now time.Time) (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate device token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(secret)
	digest := sha256.Sum256([]byte(token))
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE device SET device_token_hash=?, updated_at=? WHERE id=?`,
		hex.EncodeToString(digest[:]), formatTime(now), deviceID)
	if err != nil {
		return "", fmt.Errorf("store device token: %w", err)
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return "", ErrNotFound
	}
	if err := insertAudit(ctx, tx, deviceID, "device_token_issued", map[string]string{"status": "replaced"}, now); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return token, nil
}

func (s *Store) AuthenticateDevice(ctx context.Context, deviceID, token string) error {
	if deviceID == "" || token == "" {
		return ErrInvalidDeviceCredentials
	}
	var stored string
	err := s.db.QueryRowContext(ctx, `SELECT device_token_hash FROM device WHERE id=?`, deviceID).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) || stored == "" {
		return ErrInvalidDeviceCredentials
	}
	if err != nil {
		return fmt.Errorf("load device credential: %w", err)
	}
	digest := sha256.Sum256([]byte(token))
	provided := hex.EncodeToString(digest[:])
	if len(stored) != len(provided) || subtle.ConstantTimeCompare([]byte(stored), []byte(provided)) != 1 {
		return ErrInvalidDeviceCredentials
	}
	return nil
}

func (s *Store) ReceiveHeartbeat(ctx context.Context, deviceID string, request protocol.HeartbeatRequest, now time.Time) (protocol.HeartbeatResponse, error) {
	if request.PolicyRevision < 0 || request.SecondsUsed < 0 || now.IsZero() {
		return protocol.HeartbeatResponse{}, errors.New("invalid heartbeat counters")
	}
	parsedDate, err := time.Parse("2006-01-02", request.LocalDate)
	if err != nil || parsedDate.Format("2006-01-02") != request.LocalDate {
		return protocol.HeartbeatResponse{}, errors.New("invalid heartbeat local date")
	}
	if len(request.Events) > 100 || len(request.CommandAcks) > 100 {
		return protocol.HeartbeatResponse{}, errors.New("heartbeat batch exceeds 100 items")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	defer tx.Rollback()
	var serverRevision int64
	if err := tx.QueryRowContext(ctx, `SELECT policy_revision FROM device WHERE id=?`, deviceID).Scan(&serverRevision); errors.Is(err, sql.ErrNoRows) {
		return protocol.HeartbeatResponse{}, ErrNotFound
	} else if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	if request.PolicyRevision > serverRevision {
		return protocol.HeartbeatResponse{}, fmt.Errorf("%w: client=%d server=%d", ErrRevisionAhead, request.PolicyRevision, serverRevision)
	}
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `
		UPDATE device SET last_seen_at=?, applied_policy_revision=?, updated_at=? WHERE id=?`,
		stamp, request.PolicyRevision, stamp, deviceID); err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO daily_usage(device_id, local_date, seconds_used, last_sync_at) VALUES (?, ?, ?, ?)
		ON CONFLICT(device_id, local_date) DO UPDATE SET
			seconds_used=MAX(daily_usage.seconds_used, excluded.seconds_used),
			last_sync_at=excluded.last_sync_at`, deviceID, request.LocalDate, request.SecondsUsed, stamp); err != nil {
		return protocol.HeartbeatResponse{}, fmt.Errorf("store heartbeat usage: %w", err)
	}

	acknowledged := make([]string, 0, len(request.Events))
	for _, event := range request.Events {
		if err := receiveEvent(ctx, tx, deviceID, event); err != nil {
			return protocol.HeartbeatResponse{}, err
		}
		acknowledged = append(acknowledged, event.UUID)
	}
	for _, commandID := range request.CommandAcks {
		if commandID == "" {
			return protocol.HeartbeatResponse{}, errors.New("empty command acknowledgement")
		}
		if _, err := tx.ExecContext(ctx, `
			UPDATE device_command SET acknowledged_at=COALESCE(acknowledged_at, ?)
			WHERE id=? AND device_id=?`, stamp, commandID, deviceID); err != nil {
			return protocol.HeartbeatResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return protocol.HeartbeatResponse{}, err
	}

	response := protocol.HeartbeatResponse{ServerTime: now.UTC(), AcknowledgedEvents: acknowledged}
	if request.PolicyRevision < serverRevision {
		policy, err := s.loadPolicy(ctx, deviceID)
		if err != nil {
			return protocol.HeartbeatResponse{}, err
		}
		converted := policyForAPI(policy)
		response.Policy = &converted
	}
	response.Commands, err = s.pendingCommands(ctx, deviceID, 100)
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	return response, nil
}

func receiveEvent(ctx context.Context, tx *sql.Tx, deviceID string, event protocol.PendingEvent) error {
	if event.UUID == "" || event.Kind != "bonus_added" || event.CreatedAt.IsZero() || len(event.Payload) == 0 || len(event.Payload) > 16<<10 || !json.Valid(event.Payload) {
		return errors.New("invalid pending event")
	}
	var bonus protocol.BonusPayload
	if err := json.Unmarshal(event.Payload, &bonus); err != nil || bonus.LocalDate == "" || bonus.Seconds <= 0 || bonus.Origin != "local" {
		return errors.New("invalid local bonus event")
	}
	parsed, err := time.Parse("2006-01-02", bonus.LocalDate)
	if err != nil || parsed.Format("2006-01-02") != bonus.LocalDate {
		return errors.New("invalid bonus local date")
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO bonus(uuid, device_id, local_date, seconds, origin, created_at)
		VALUES (?, ?, ?, ?, 'local', ?)`, event.UUID, deviceID, bonus.LocalDate, bonus.Seconds, formatTime(event.CreatedAt)); err != nil {
		return fmt.Errorf("store local bonus event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT OR IGNORE INTO audit_event(uuid, device_id, kind, origin, payload_json, created_at)
		VALUES (?, ?, 'bonus_added', 'local', ?, ?)`, event.UUID, deviceID, string(event.Payload), formatTime(event.CreatedAt)); err != nil {
		return fmt.Errorf("audit local bonus event: %w", err)
	}
	return nil
}

func (s *Store) pendingCommands(ctx context.Context, deviceID string, limit int) ([]protocol.Command, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, kind, payload_json, created_at FROM device_command
		WHERE device_id=? AND acknowledged_at IS NULL ORDER BY created_at, id LIMIT ?`, deviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var commands []protocol.Command
	for rows.Next() {
		var command protocol.Command
		var payload, created string
		if err := rows.Scan(&command.ID, &command.Kind, &payload, &created); err != nil {
			return nil, err
		}
		command.Payload = json.RawMessage(payload)
		command.CreatedAt, err = parseTime(created)
		if err != nil {
			return nil, err
		}
		commands = append(commands, command)
	}
	return commands, rows.Err()
}

func policyForAPI(policy Policy) protocol.Policy {
	converted := protocol.Policy{
		Revision: policy.Revision, MonitoringPaused: policy.MonitoringPaused,
		ManualBlock: policy.ManualBlock, WarningMinutes: policy.WarningMinutes,
		LocalPasswordVerifier: policy.LocalPasswordVerifier,
		WeeklyQuotaSeconds:    policy.WeeklyQuota, UpdatedAt: policy.UpdatedAt,
	}
	for _, routine := range policy.Routines {
		converted.Routines = append(converted.Routines, protocol.Routine{
			ID: routine.ID, Name: routine.Name, Days: routine.Days,
			StartSecond: routine.Start, EndSecond: routine.End, Enabled: routine.Enabled,
		})
	}
	return converted
}

// QueueControl updates the authoritative policy and queues an at-least-once command.
func (s *Store) QueueControl(ctx context.Context, deviceID, kind string, now time.Time) error {
	setClause := ""
	switch kind {
	case "pause_monitoring":
		setClause = "monitoring_paused=1"
	case "resume_monitoring":
		setClause = "monitoring_paused=0"
	case "block_now":
		setClause = "monitoring_paused=0, manual_block=1"
	case "clear_manual_block":
		setClause = "manual_block=0"
	default:
		return errors.New("unknown device control")
	}
	commandID, err := newID()
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stamp := formatTime(now)
	result, err := tx.ExecContext(ctx, `UPDATE policy SET `+setClause+`, revision=revision+1, updated_at=? WHERE device_id=?`, stamp, deviceID)
	if err != nil {
		return err
	}
	changed, _ := result.RowsAffected()
	if changed == 0 {
		return ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `UPDATE device SET policy_revision=policy_revision+1, updated_at=? WHERE id=?`, stamp, deviceID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO device_command(id, device_id, kind, payload_json, created_at) VALUES (?, ?, ?, '{}', ?)`, commandID, deviceID, kind, stamp); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, deviceID, kind, map[string]string{"status": "queued"}, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) QueueRemoteBonus(ctx context.Context, deviceID, localDate string, seconds int64, now time.Time) error {
	parsed, err := time.Parse("2006-01-02", localDate)
	if err != nil || parsed.Format("2006-01-02") != localDate || seconds <= 0 || seconds > int64((24*time.Hour)/time.Second) {
		return errors.New("invalid remote bonus")
	}
	uuid, err := newID()
	if err != nil {
		return err
	}
	payload, err := json.Marshal(protocol.BonusPayload{UUID: uuid, LocalDate: localDate, Seconds: seconds, Origin: "web", CreatedAt: now.UTC()})
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stamp := formatTime(now)
	if _, err := tx.ExecContext(ctx, `INSERT INTO bonus(uuid, device_id, local_date, seconds, origin, created_at) VALUES (?, ?, ?, ?, 'web', ?)`, uuid, deviceID, localDate, seconds, stamp); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO device_command(id, device_id, kind, payload_json, created_at) VALUES (?, ?, 'add_bonus', ?, ?)`, uuid, deviceID, string(payload), stamp); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, deviceID, "bonus_added", map[string]interface{}{"origin": "web", "seconds": seconds, "local_date": localDate}, now); err != nil {
		return err
	}
	return tx.Commit()
}
