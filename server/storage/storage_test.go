package storage

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAdministrativePolicyLifecycle(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)

	created, err := store.BootstrapAdmin(ctx, "admin", "hash", now)
	if err != nil || !created {
		t.Fatalf("bootstrap created=%t err=%v", created, err)
	}
	created, err = store.BootstrapAdmin(ctx, "other", "other-hash", now)
	if err != nil || created {
		t.Fatalf("second bootstrap created=%t err=%v", created, err)
	}
	admin, err := store.AdminByLogin(ctx, "admin")
	if err != nil || !admin.Active || admin.PasswordHash != "hash" {
		t.Fatalf("admin=%+v err=%v", admin, err)
	}

	device, err := store.CreateDevice(ctx, "PC do quarto", now)
	if err != nil {
		t.Fatal(err)
	}
	var quotas [7]int64
	quotas[time.Monday] = 2 * 60 * 60
	quotas[time.Tuesday] = 45 * 60
	if err := store.SaveQuotas(ctx, device.ID, quotas, 10, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	_, policy, err := store.LoadDevice(ctx, device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if policy.WeeklyQuota[time.Monday] != 7200 || policy.WeeklyQuota[time.Tuesday] != 2700 {
		t.Fatalf("independent quotas=%v", policy.WeeklyQuota)
	}

	var weekdays [7]bool
	for day := time.Monday; day <= time.Friday; day++ {
		weekdays[day] = true
	}
	routineID, err := store.SaveRoutine(ctx, device.ID, Routine{
		Name: "Dormir", Days: weekdays, Start: 22 * 60 * 60, End: 8 * 60 * 60, Enabled: true,
	}, now.Add(2*time.Minute))
	if err != nil || routineID == "" {
		t.Fatalf("routine id=%q err=%v", routineID, err)
	}
	_, policy, err = store.LoadDevice(ctx, device.ID)
	if err != nil || len(policy.Routines) != 1 {
		t.Fatalf("policy=%+v err=%v", policy, err)
	}
	routine := policy.Routines[0]
	if routine.Start != 79200 || routine.End != 28800 || !routine.Days[time.Monday] || routine.Days[time.Saturday] {
		t.Fatalf("overnight weekday routine=%+v", routine)
	}

	verifier := "$argon2id$v=19$m=8192,t=1,p=1$c2FsdHNhbHQ$aGFzaGhhc2hoYXNoaGFzaA"
	if err := store.SetLocalPassword(ctx, device.ID, verifier, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	events, err := store.ListAudit(ctx, device.ID, 20)
	if err != nil {
		t.Fatal(err)
	}
	kinds := make(map[string]bool)
	for _, event := range events {
		kinds[event.Kind] = true
		if strings.Contains(event.Details, "argon2") || strings.Contains(event.Details, verifier) {
			t.Fatalf("audit leaked password verifier: %+v", event)
		}
	}
	for _, kind := range []string{"device_created", "quotas_updated", "routine_saved", "local_password_changed"} {
		if !kinds[kind] {
			t.Fatalf("audit missing %s: %+v", kind, events)
		}
	}
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(context.Background(), filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}
