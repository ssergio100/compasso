package pamgate

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/policy"
	"github.com/sergio/compasso/agent/storage"
)

func TestCheckOnlyControlsConfiguredUser(t *testing.T) {
	store := openStore(t)
	defer store.Close()
	result, err := Check(context.Background(), store, "child", "parent", time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !result.Allowed {
		t.Fatalf("uncontrolled account was denied: %+v", result)
	}
}

func TestCheckAllowsAndDeniesFromDurablePolicy(t *testing.T) {
	ctx := context.Background()
	store := openStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	var quotas [7]time.Duration
	quotas[now.Weekday()] = 10 * time.Second
	snapshot := storage.PolicySnapshot{Revision: 1, WeeklyQuota: quotas, UpdatedAt: now}
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}

	result, err := Check(ctx, store, "child", "child", now)
	if err != nil || !result.Allowed {
		t.Fatalf("allowed result=%+v err=%v", result, err)
	}
	if err := store.CheckpointUsage(ctx, storage.DailyUsage{
		LocalDate: now.Format("2006-01-02"), SecondsUsed: 10, CheckpointAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	result, err = Check(ctx, store, "child", "child", now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Allowed || result.Reason != policy.ReasonQuota {
		t.Fatalf("exhausted result=%+v", result)
	}
}

func TestCheckHonorsPauseAndRoutine(t *testing.T) {
	ctx := context.Background()
	store := openStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 22, 30, 0, 0, time.Local)
	var quotas [7]time.Duration
	quotas[now.Weekday()] = time.Hour
	var days [7]bool
	days[now.Weekday()] = true
	snapshot := storage.PolicySnapshot{
		Revision: 1, WeeklyQuota: quotas, UpdatedAt: now,
		Routines: []storage.StoredRoutine{{
			ID: "sleep", Name: "Sleep", Days: days,
			Start: 22 * time.Hour, End: 8 * time.Hour, Enabled: true,
		}},
	}
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	result, err := Check(ctx, store, "child", "child", now)
	if err != nil || result.Reason != policy.ReasonRoutine || result.Allowed {
		t.Fatalf("routine result=%+v err=%v", result, err)
	}

	snapshot.Revision = 2
	snapshot.MonitoringPaused = true
	snapshot.UpdatedAt = now.Add(time.Second)
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	result, err = Check(ctx, store, "child", "child", now)
	if err != nil || !result.Allowed || result.Reason != policy.ReasonPaused {
		t.Fatalf("paused result=%+v err=%v", result, err)
	}
}

func openStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}
