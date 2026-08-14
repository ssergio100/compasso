package storage

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestBindEnrollmentClearsUnconfirmedLegacyState(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "agent.db"))
	defer store.Close()
	if err := store.ReplacePolicy(ctx, samplePolicy(6)); err != nil {
		t.Fatal(err)
	}
	if err := store.CheckpointUsage(ctx, DailyUsage{
		LocalDate: "2026-08-12", SecondsUsed: 300, CheckpointAt: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	reset, err := store.BindEnrollment(ctx, "https://api.example.test", "new-device", false)
	if err != nil || !reset {
		t.Fatalf("reset=%t err=%v", reset, err)
	}
	if _, err := store.CurrentPolicy(); !errors.Is(err, ErrNoPolicy) {
		t.Fatalf("old policy remains available: %v", err)
	}
	usage, err := store.LoadDailyUsage(ctx, "2026-08-12")
	if err != nil || usage.SecondsUsed != 0 {
		t.Fatalf("old usage=%+v err=%v", usage, err)
	}
	if err := store.ReplacePolicy(ctx, samplePolicy(4)); err != nil {
		t.Fatalf("new server revision was rejected after reset: %v", err)
	}
}

func TestBindEnrollmentPreservesConfirmedUpgradeAndSameDevice(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "agent.db"))
	defer store.Close()
	if err := store.ReplacePolicy(ctx, samplePolicy(6)); err != nil {
		t.Fatal(err)
	}
	reset, err := store.BindEnrollment(ctx, "https://api.example.test", "device-1", true)
	if err != nil || reset {
		t.Fatalf("confirmed legacy binding reset=%t err=%v", reset, err)
	}
	reset, err = store.BindEnrollment(ctx, "https://api.example.test", "device-1", false)
	if err != nil || reset {
		t.Fatalf("same enrollment reset=%t err=%v", reset, err)
	}
	policy, err := store.CurrentPolicy()
	if err != nil || policy.Revision != 6 {
		t.Fatalf("preserved policy=%+v err=%v", policy, err)
	}
}

func TestBindEnrollmentClearsStateWhenDeviceChanges(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "agent.db"))
	defer store.Close()
	if _, err := store.BindEnrollment(ctx, "https://api.example.test", "device-1", true); err != nil {
		t.Fatal(err)
	}
	if err := store.ReplacePolicy(ctx, samplePolicy(6)); err != nil {
		t.Fatal(err)
	}
	reset, err := store.BindEnrollment(ctx, "https://api.example.test", "device-2", true)
	if err != nil || !reset {
		t.Fatalf("changed enrollment reset=%t err=%v", reset, err)
	}
	if _, err := store.CurrentPolicy(); !errors.Is(err, ErrNoPolicy) {
		t.Fatalf("old device policy remains available: %v", err)
	}
}
