// Package syncclient exchanges durable agent state with tempo-server.
package syncclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sergio/compasso/agent/storage"
	protocol "github.com/sergio/compasso/protocol/v1"
)

type Config struct {
	ServerURL         string
	DeviceID          string
	DeviceToken       string
	HeartbeatInterval time.Duration
}

type Client struct {
	store  *storage.Store
	http   *http.Client
	config Config
	now    func() time.Time
}

func New(store *storage.Store, httpClient *http.Client, config Config) (*Client, error) {
	if store == nil || httpClient == nil {
		return nil, errors.New("store and HTTP client are required")
	}
	if config.ServerURL == "" || config.DeviceID == "" || config.DeviceToken == "" || config.HeartbeatInterval <= 0 {
		return nil, errors.New("complete synchronization configuration is required")
	}
	config.ServerURL = strings.TrimRight(config.ServerURL, "/")
	return &Client{store: store, http: httpClient, config: config, now: time.Now}, nil
}

// Heartbeat performs one cycle. Durable events are removed only when their
// identifiers are explicitly acknowledged by the server.
func (c *Client) Heartbeat(ctx context.Context, now time.Time) (protocol.HeartbeatResponse, error) {
	if now.IsZero() {
		return protocol.HeartbeatResponse{}, errors.New("heartbeat time is required")
	}
	revision := int64(0)
	policy, err := c.store.LoadPolicy(ctx)
	if err == nil {
		revision = policy.Revision
	} else if !errors.Is(err, storage.ErrNoPolicy) {
		return protocol.HeartbeatResponse{}, err
	}
	localDate := now.Format("2006-01-02")
	usage, err := c.store.LoadDailyUsage(ctx, localDate)
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	pending, err := c.store.PendingEvents(ctx, 100)
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	commandAcks, err := c.store.AppliedCommandIDs(ctx, 100)
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	request := protocol.HeartbeatRequest{
		PolicyRevision: revision, LocalDate: localDate, SecondsUsed: usage.SecondsUsed,
		CommandAcks: commandAcks,
	}
	for _, event := range pending {
		if !json.Valid([]byte(event.PayloadJSON)) {
			_ = c.store.IncrementEventRetry(ctx, event.UUID)
			return protocol.HeartbeatResponse{}, fmt.Errorf("pending event %s contains invalid JSON", event.UUID)
		}
		request.Events = append(request.Events, protocol.PendingEvent{
			UUID: event.UUID, Kind: event.Kind, Payload: json.RawMessage(event.PayloadJSON), CreatedAt: event.CreatedAt,
		})
	}
	body, err := json.Marshal(request)
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.config.ServerURL+protocol.HeartbeatPath, bytes.NewReader(body))
	if err != nil {
		return protocol.HeartbeatResponse{}, err
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Authorization", "Bearer "+c.config.DeviceToken)
	httpRequest.Header.Set("X-Tempo-Device-ID", c.config.DeviceID)
	response, err := c.http.Do(httpRequest)
	if err != nil {
		c.recordRetries(ctx, pending)
		return protocol.HeartbeatResponse{}, fmt.Errorf("send heartbeat: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		c.recordRetries(ctx, pending)
		return protocol.HeartbeatResponse{}, fmt.Errorf("heartbeat returned HTTP %d", response.StatusCode)
	}
	var result protocol.HeartbeatResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 512<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		c.recordRetries(ctx, pending)
		return protocol.HeartbeatResponse{}, fmt.Errorf("decode heartbeat response: %w", err)
	}
	for _, eventID := range result.AcknowledgedEvents {
		if err := c.store.AcknowledgeEvent(ctx, eventID); err != nil {
			return protocol.HeartbeatResponse{}, err
		}
	}
	if result.Policy != nil {
		if err := c.store.ReplacePolicy(ctx, fromProtocolPolicy(*result.Policy)); err != nil {
			return protocol.HeartbeatResponse{}, fmt.Errorf("apply policy revision %d: %w", result.Policy.Revision, err)
		}
	}
	for _, command := range result.Commands {
		if err := c.applyCommand(ctx, command, now); err != nil {
			return protocol.HeartbeatResponse{}, fmt.Errorf("apply command %s: %w", command.ID, err)
		}
	}
	return result, nil
}

func (c *Client) applyCommand(ctx context.Context, command protocol.Command, now time.Time) error {
	if command.ID == "" || command.CreatedAt.IsZero() {
		return errors.New("invalid command envelope")
	}
	switch command.Kind {
	case "pause_monitoring", "resume_monitoring", "block_now", "clear_manual_block":
		// These states are carried by the authoritative policy revision. Recording
		// the command makes delivery idempotent and acknowledges it next cycle.
		return c.store.RecordCommandApplied(ctx, command.ID, now)
	case "add_bonus":
		var payload protocol.BonusPayload
		if err := json.Unmarshal(command.Payload, &payload); err != nil {
			return errors.New("invalid bonus command payload")
		}
		if payload.UUID == "" {
			payload.UUID = command.ID
		}
		if payload.CreatedAt.IsZero() {
			payload.CreatedAt = command.CreatedAt
		}
		return c.store.ApplyRemoteBonus(ctx, command.ID, storage.Bonus{
			UUID: payload.UUID, LocalDate: payload.LocalDate, Seconds: payload.Seconds,
			Origin: "web", CreatedAt: payload.CreatedAt,
		}, now)
	default:
		return fmt.Errorf("unsupported command kind %q", command.Kind)
	}
}

func (c *Client) recordRetries(ctx context.Context, events []storage.PendingEvent) {
	for _, event := range events {
		_ = c.store.IncrementEventRetry(ctx, event.UUID)
	}
}

func (c *Client) Run(ctx context.Context, logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}
	delay := time.Duration(0)
	backoff := time.Second
	online := false
	first := true
	for {
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil
			case <-timer.C:
			}
		}
		_, err := c.Heartbeat(ctx, c.now())
		if err != nil {
			if online || first {
				logger.Printf("synchronization offline: %v", err)
			}
			online, first = false, false
			delay = backoff
			backoff *= 2
			if backoff > c.config.HeartbeatInterval {
				backoff = c.config.HeartbeatInterval
			}
			continue
		}
		if !online {
			logger.Printf("synchronization online")
		}
		online, first = true, false
		backoff = time.Second
		delay = c.config.HeartbeatInterval
	}
}

func fromProtocolPolicy(policy protocol.Policy) storage.PolicySnapshot {
	converted := storage.PolicySnapshot{
		Revision: policy.Revision, MonitoringPaused: policy.MonitoringPaused,
		ManualBlock: policy.ManualBlock, WarningMinutes: policy.WarningMinutes,
		LocalPasswordVerifier: policy.LocalPasswordVerifier, UpdatedAt: policy.UpdatedAt,
	}
	for day, seconds := range policy.WeeklyQuotaSeconds {
		converted.WeeklyQuota[day] = time.Duration(seconds) * time.Second
	}
	for _, routine := range policy.Routines {
		converted.Routines = append(converted.Routines, storage.StoredRoutine{
			ID: routine.ID, Name: routine.Name, Days: routine.Days,
			Start: time.Duration(routine.StartSecond) * time.Second,
			End:   time.Duration(routine.EndSecond) * time.Second, Enabled: routine.Enabled,
		})
	}
	return converted
}
