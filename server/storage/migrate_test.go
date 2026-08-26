package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func TestMigrationElevenAddsAndBackfillsVisualIdentities(t *testing.T) {
	ctx := context.Background()
	databasePath := filepath.Join(t.TempDir(), "server.db")
	db, err := sql.Open("sqlite3", databasePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE device (id TEXT PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE routine (id TEXT PRIMARY KEY, device_id TEXT NOT NULL, name TEXT NOT NULL);
		INSERT INTO schema_migrations(version) VALUES (1),(2),(3),(4),(5),(6),(7),(8),(9),(10);
		INSERT INTO device(id,name) VALUES ('device-1','PC antigo');
		INSERT INTO routine(id,device_id,name) VALUES ('sleep','device-1','Hora de dormir'),('reading','device-1','Leitura');
	`); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := Open(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var avatarKey, sleepIcon, readingIcon string
	if err := store.db.QueryRowContext(ctx, `SELECT avatar_key FROM device WHERE id='device-1'`).Scan(&avatarKey); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT icon_key FROM routine WHERE id='sleep'`).Scan(&sleepIcon); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT icon_key FROM routine WHERE id='reading'`).Scan(&readingIcon); err != nil {
		t.Fatal(err)
	}
	if avatarKey != "cat" || sleepIcon != "sleep" || readingIcon != "reading" {
		t.Fatalf("backfill avatar=%q sleep=%q reading=%q", avatarKey, sleepIcon, readingIcon)
	}
}

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
