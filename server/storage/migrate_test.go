package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrationNineRepairsActivitySchemaRecordedButMissing(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "server.db")
	store, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 24, 3, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Servidor parcialmente migrado", now)
	if err != nil {
		t.Fatal(err)
	}
	commandID, err := store.QueueRemoteBonus(ctx, device.ID, 15*60, now)
	if err != nil {
		t.Fatal(err)
	}

	// Reproduce the production state: version 8 was recorded although its
	// human-history tables were not present. Version 9 must rebuild them from
	// the durable command and audit records.
	if _, err := store.db.ExecContext(ctx, `
		DROP TABLE activity_step;
		DROP TABLE activity;
		DROP TABLE activity_history_maintenance;
		DELETE FROM schema_migrations WHERE version IN (9, 10);
	`); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	repaired, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatalf("open did not repair partial activity migration: %v", err)
	}
	defer repaired.Close()
	activity, err := repaired.LoadDeviceActivity(ctx, device.ID, commandID)
	if err != nil || activity.Status != "waiting_device" || findActivityStep(activity.Steps, "requested") == nil {
		t.Fatalf("reconstructed activity=%+v err=%v", activity, err)
	}
	var version int
	if err := repaired.db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&version); err != nil || version != 10 {
		t.Fatalf("repaired user_version=%d err=%v", version, err)
	}
}

func TestMigrationTenBackfillsExistingAdministrativeAudit(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "server.db")
	store, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 24, 3, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.db.ExecContext(ctx, `
		DELETE FROM activity;
		INSERT INTO audit_event(uuid, device_id, kind, origin, payload_json, created_at)
		VALUES ('historical-policy-change', ?, 'quotas_updated', 'web',
		        '{"warning_minutes":15}', ?);
		DELETE FROM schema_migrations WHERE version=10;
	`, device.ID, formatTime(now.Add(time.Minute))); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	activities, err := reopened.ListDeviceActivities(ctx, device.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, activity := range activities {
		counts[activity.Kind]++
	}
	if counts["device_created"] != 1 || counts["quotas_updated"] != 1 {
		t.Fatalf("historical activities were not backfilled: %+v", activities)
	}
}
