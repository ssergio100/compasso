package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	protocol "github.com/sergio/compasso/protocol/v1"
)

func TestDeviceAuthenticationAndDuplicateHeartbeatAreIdempotent(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	token, err := store.IssueDeviceToken(ctx, device.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AuthenticateDevice(ctx, device.ID, token); err != nil {
		t.Fatalf("valid device credential rejected: %v", err)
	}
	if err := store.AuthenticateDevice(ctx, device.ID, "wrong"); !errors.Is(err, ErrInvalidDeviceCredentials) {
		t.Fatalf("wrong credential error=%v", err)
	}
	if err := store.RevokeDeviceToken(ctx, device.ID, now.Add(time.Millisecond)); err != nil {
		t.Fatal(err)
	}
	if err := store.AuthenticateDevice(ctx, device.ID, token); !errors.Is(err, ErrInvalidDeviceCredentials) {
		t.Fatalf("revoked credential error=%v", err)
	}
	token, err = store.IssueDeviceToken(ctx, device.ID, now.Add(2*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(protocol.BonusPayload{LocalDate: "2026-08-10", Seconds: 300, Origin: "local"})
	heartbeat := protocol.HeartbeatRequest{
		PolicyRevision: 0, LocalDate: "2026-08-10", SecondsUsed: 120,
		Events: []protocol.PendingEvent{{
			UUID: "local-bonus-1", Kind: "bonus_added", Payload: payload, CreatedAt: now,
		}},
	}
	first, err := store.ReceiveHeartbeat(ctx, device.ID, heartbeat, now.Add(time.Second))
	if err != nil || first.Policy == nil || first.Policy.Revision != 1 || len(first.AcknowledgedEvents) != 1 {
		t.Fatalf("first heartbeat response=%+v err=%v", first, err)
	}
	heartbeat.PolicyRevision = 1
	heartbeat.SecondsUsed = 60 // An older absolute checkpoint cannot reduce usage.
	second, err := store.ReceiveHeartbeat(ctx, device.ID, heartbeat, now.Add(2*time.Second))
	if err != nil || second.Policy != nil || len(second.AcknowledgedEvents) != 1 {
		t.Fatalf("duplicate heartbeat response=%+v err=%v", second, err)
	}
	summary, err := store.LoadDailySummary(ctx, device.ID, "2026-08-10")
	if err != nil || summary.UsedSeconds != 120 || summary.BonusSeconds != 300 {
		t.Fatalf("duplicate changed summary=%+v err=%v", summary, err)
	}
	latestLocalDate, err := store.LatestHeartbeatLocalDate(ctx, device.ID)
	if err != nil || latestLocalDate != "2026-08-10" {
		t.Fatalf("latest heartbeat local date=%q err=%v", latestLocalDate, err)
	}
	events, err := store.ListAudit(ctx, device.ID, 50)
	if err != nil {
		t.Fatal(err)
	}
	bonusEvents := 0
	for _, event := range events {
		if event.Kind == "bonus_added" && event.Origin == "local" {
			bonusEvents++
		}
	}
	if bonusEvents != 1 {
		t.Fatalf("duplicate heartbeat created %d bonus audit events", bonusEvents)
	}
}

func TestCommandsRemainPendingUntilAcknowledged(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.QueueRemoteBonus(ctx, device.ID, "2026-08-10", 600, now); err != nil {
		t.Fatal(err)
	}
	request := protocol.HeartbeatRequest{PolicyRevision: 1, LocalDate: "2026-08-10"}
	first, err := store.ReceiveHeartbeat(ctx, device.ID, request, now.Add(time.Second))
	if err != nil || len(first.Commands) != 1 || first.Commands[0].Kind != "add_bonus" {
		t.Fatalf("pending commands=%+v err=%v", first.Commands, err)
	}
	second, err := store.ReceiveHeartbeat(ctx, device.ID, request, now.Add(2*time.Second))
	if err != nil || len(second.Commands) != 1 {
		t.Fatalf("unacknowledged command was not redelivered: %+v err=%v", second.Commands, err)
	}
	request.CommandAcks = []string{first.Commands[0].ID}
	third, err := store.ReceiveHeartbeat(ctx, device.ID, request, now.Add(3*time.Second))
	if err != nil || len(third.Commands) != 0 {
		t.Fatalf("acknowledged command still pending: %+v err=%v", third.Commands, err)
	}
}

func TestHeartbeatRejectsUnexpectedAuditFields(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"local_date":"2026-08-10","seconds":300,"origin":"local","device_token":"must-not-be-stored"}`)
	_, err = store.ReceiveHeartbeat(ctx, device.ID, protocol.HeartbeatRequest{
		PolicyRevision: 1, LocalDate: "2026-08-10",
		Events: []protocol.PendingEvent{{UUID: "unexpected-field", Kind: "bonus_added", Payload: payload, CreatedAt: now}},
	}, now)
	if err == nil {
		t.Fatal("event with an unexpected sensitive field was accepted")
	}
	events, listError := store.ListAudit(ctx, device.ID, 20)
	if listError != nil {
		t.Fatal(listError)
	}
	for _, event := range events {
		if strings.Contains(event.Details, "must-not-be-stored") {
			t.Fatalf("audit stored a sensitive field: %+v", event)
		}
	}
}
