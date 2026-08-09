package localauth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/storage"
)

func TestCorrectPasswordAddsExactIdempotentBonusEvent(t *testing.T) {
	ctx := context.Background()
	store, service, now := testService(t)
	defer store.Close()

	result, err := service.Grant(ctx, "secret", 30*60, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.BonusSeconds != 1800 || result.TotalSeconds != 1800 || result.UUID == "" {
		t.Fatalf("unexpected grant result: %+v", result)
	}
	events, err := store.PendingEvents(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].UUID != result.UUID || events[0].Kind != "bonus_added" {
		t.Fatalf("pending events = %+v", events)
	}
}

func TestWrongPasswordDoesNotAddBonusAndRateLimits(t *testing.T) {
	ctx := context.Background()
	store, service, now := testService(t)
	defer store.Close()

	if _, err := service.Grant(ctx, "wrong", 15*60, now); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("wrong password error=%v", err)
	}
	if _, err := service.Grant(ctx, "secret", 15*60, now.Add(time.Second)); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("early retry error=%v", err)
	}
	total, err := store.TotalBonusSeconds(ctx, now.Format("2006-01-02"))
	if err != nil || total != 0 {
		t.Fatalf("bonus after failures=%d err=%v", total, err)
	}
	if _, err := service.Grant(ctx, "secret", 15*60, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
}

func TestMissingPasswordIsReportedExplicitly(t *testing.T) {
	ctx := context.Background()
	store, err := storage.Open(ctx, filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, storage.PolicySnapshot{Revision: 1, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Grant(ctx, "anything", 15*60, now); !errors.Is(err, ErrPasswordNotConfigured) {
		t.Fatalf("missing password error=%v", err)
	}
}

func TestBonusSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	directory := t.TempDir()
	path := filepath.Join(directory, "agent.db")
	now := time.Date(2026, time.August, 10, 22, 30, 0, 0, time.Local)
	store, service := serviceAtPath(t, path, now)
	if _, err := service.Grant(ctx, "secret", 60*60, now); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	store, err := storage.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	total, err := store.TotalBonusSeconds(ctx, now.Format("2006-01-02"))
	if err != nil || total != 3600 {
		t.Fatalf("bonus after restart=%d err=%v", total, err)
	}
}

func testService(t *testing.T) (*storage.Store, *Service, time.Time) {
	t.Helper()
	now := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	store, service := serviceAtPath(t, filepath.Join(t.TempDir(), "agent.db"), now)
	return store, service, now
}

func serviceAtPath(t *testing.T, path string, now time.Time) (*storage.Store, *Service) {
	t.Helper()
	ctx := context.Background()
	store, err := storage.Open(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := HashPassword("secret", testParams)
	if err != nil {
		t.Fatal(err)
	}
	var quotas [7]time.Duration
	quotas[now.Weekday()] = time.Hour
	snapshot := storage.PolicySnapshot{
		Revision: 1, WeeklyQuota: quotas, LocalPasswordVerifier: verifier, UpdatedAt: now,
	}
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	return store, service
}
