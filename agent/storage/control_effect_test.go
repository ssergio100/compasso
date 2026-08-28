package storage

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestControlEffectBecomesAcknowledgableOnlyAfterCompletion(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "agent.db"))
	defer store.Close()
	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	if err := store.StageControlEffect(ctx, "block-1", 2, "block_now", now); err != nil {
		t.Fatal(err)
	}
	if ids, err := store.AppliedCommandIDs(ctx, 10); err != nil || len(ids) != 0 {
		t.Fatalf("unconfirmed effect acknowledgements=%v err=%v", ids, err)
	}
	effect, available, err := store.PendingControlEffect(ctx)
	if err != nil || !available || effect.CommandID != "block-1" || effect.Revision != 2 || effect.Kind != "block_now" {
		t.Fatalf("pending effect=%+v available=%t err=%v", effect, available, err)
	}
	if err := store.CompleteControlEffect(ctx, "block-1", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, available, err := store.PendingControlEffect(ctx); err != nil || available {
		t.Fatalf("completed effect remained available=%t err=%v", available, err)
	}
	ids, err := store.AppliedCommandIDs(ctx, 10)
	if err != nil || len(ids) != 1 || ids[0] != "block-1" {
		t.Fatalf("completed effect acknowledgements=%v err=%v", ids, err)
	}
	if err := store.ForgetAppliedCommandIDs(ctx, ids); err != nil {
		t.Fatal(err)
	}
	if ids, err := store.AppliedCommandIDs(ctx, 10); err != nil || len(ids) != 0 {
		t.Fatalf("server-confirmed acknowledgement remained=%v err=%v", ids, err)
	}
}

func TestNewControlEffectSupersedesOlderLocalTransition(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "agent.db"))
	defer store.Close()
	now := time.Date(2026, time.August, 27, 12, 0, 0, 0, time.UTC)
	if err := store.StageControlEffect(ctx, "block-1", 2, "block_now", now); err != nil {
		t.Fatal(err)
	}
	if err := store.StageControlEffect(ctx, "clear-2", 3, "clear_manual_block", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	effect, available, err := store.PendingControlEffect(ctx)
	if err != nil || !available || effect.CommandID != "clear-2" || effect.Revision != 3 || effect.Kind != "clear_manual_block" {
		t.Fatalf("latest pending effect=%+v available=%t err=%v", effect, available, err)
	}
}
