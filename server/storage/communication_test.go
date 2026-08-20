package storage

import (
	"context"
	"testing"
	"time"
)

func TestCommunicationLogsAreSanitizedRetainedAndDeleted(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 14, 12, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	event := CommunicationLog{
		DeviceID: device.ID, Source: "agent", Target: "api", Operation: "heartbeat",
		Result: "success", HTTPStatus: 200, DurationMS: 84, Summary: "Heartbeat processado.",
		Details: map[string]string{"protocol_version": "2"},
	}
	if _, err := store.AppendCommunicationLog(ctx, event, now.AddDate(0, 0, -40)); err != nil {
		t.Fatal(err)
	}
	stored, err := store.AppendCommunicationLog(ctx, event, now)
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.ListCommunicationLogs(ctx, device.ID, 0, 20)
	if err != nil || len(events) != 1 || events[0].ID != stored.ID || events[0].Details["protocol_version"] != "2" {
		t.Fatalf("retained communication logs=%+v err=%v", events, err)
	}
	newer, err := store.AppendCommunicationLog(ctx, event, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	incremental, err := store.ListCommunicationLogs(ctx, device.ID, stored.ID, 20)
	if err != nil || len(incremental) != 1 || incremental[0].ID != newer.ID {
		t.Fatalf("incremental communication logs=%+v err=%v", incremental, err)
	}
	unsafe := event
	unsafe.Details = map[string]string{"device_token": "must-not-be-stored"}
	if _, err := store.AppendCommunicationLog(ctx, unsafe, now); err == nil {
		t.Fatal("sensitive communication detail was accepted")
	}
	if err := store.SetCommunicationRetentionDays(ctx, 7, now); err != nil {
		t.Fatal(err)
	}
	if days, err := store.CommunicationRetentionDays(ctx); err != nil || days != 7 {
		t.Fatalf("retention days=%d err=%v", days, err)
	}
	if deleted, err := store.DeleteCommunicationLogs(ctx, device.ID); err != nil || deleted != 2 {
		t.Fatalf("deleted=%d err=%v", deleted, err)
	}
}
