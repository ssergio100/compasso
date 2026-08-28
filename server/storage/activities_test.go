package storage

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	protocol "github.com/ssergio100/compasso/protocol/v1"
)

func TestCompletedActivityHistoryIsCleanedWithoutTouchingBusinessData(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now.AddDate(0, 0, -45))
	if err != nil {
		t.Fatal(err)
	}

	oldCompletedID, err := store.QueueRemoteBonus(ctx, device.ID, 15*60, now.AddDate(0, 0, -40))
	if err != nil {
		t.Fatal(err)
	}
	pendingID, err := store.QueueControlOperation(ctx, device.ID, "pause_monitoring", now.AddDate(0, 0, -39))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReceiveHeartbeat(ctx, device.ID, protocol.HeartbeatRequest{
		PolicyRevision: 0, LocalDate: "2026-07-09", CommandAcks: []string{oldCompletedID},
	}, now.AddDate(0, 0, -32)); err != nil {
		t.Fatal(err)
	}

	recentCompletedID, err := store.QueueRemoteBonus(ctx, device.ID, 10*60, now.AddDate(0, 0, -2))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.ReceiveHeartbeat(ctx, device.ID, protocol.HeartbeatRequest{
		PolicyRevision: 0, LocalDate: "2026-08-09", CommandAcks: []string{recentCompletedID},
	}, now.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}

	if _, err := store.LoadDeviceActivity(ctx, device.ID, oldCompletedID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("activity completed more than 30 days ago was retained: %v", err)
	}
	if acknowledged, err := store.RemoteBonusAcknowledged(ctx, device.ID, oldCompletedID); err != nil || !acknowledged {
		t.Fatalf("automatic history cleanup changed the real bonus command: acknowledged=%t err=%v", acknowledged, err)
	}
	if activity, err := store.LoadDeviceActivity(ctx, device.ID, pendingID); err != nil || activity.Status == "completed" {
		t.Fatalf("pending activity was removed or completed: activity=%+v err=%v", activity, err)
	}
	if activity, err := store.LoadDeviceActivity(ctx, device.ID, recentCompletedID); err != nil || activity.Status != "completed" {
		t.Fatalf("recent completed activity was removed: activity=%+v err=%v", activity, err)
	}

	hidden, err := store.DeleteCompletedDeviceActivities(ctx, device.ID)
	if err != nil || hidden != 1 {
		t.Fatalf("manual completed cleanup hidden=%d err=%v", hidden, err)
	}
	if _, err := store.LoadDeviceActivity(ctx, device.ID, recentCompletedID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("manual cleanup retained completed activity: %v", err)
	}
	if acknowledged, err := store.RemoteBonusAcknowledged(ctx, device.ID, recentCompletedID); err != nil || !acknowledged {
		t.Fatalf("manual history cleanup changed the real bonus command: acknowledged=%t err=%v", acknowledged, err)
	}
	if _, err := store.LoadDeviceActivity(ctx, device.ID, pendingID); err != nil {
		t.Fatalf("manual cleanup removed pending activity: %v", err)
	}
}

func TestLocalBonusBecomesOneHumanActivityAndSurvivesHistoryCleanup(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(protocol.BonusPayload{
		LocalDate: "2026-08-10", Seconds: 30 * 60, Origin: "local",
	})
	request := protocol.HeartbeatRequest{
		PolicyRevision: 0, LocalDate: "2026-08-10",
		Events: []protocol.PendingEvent{{
			UUID: "local-bonus-30m", Kind: "bonus_added", Payload: payload, CreatedAt: now,
		}},
	}
	if _, err := store.ReceiveHeartbeat(ctx, device.ID, request, now.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	// A retry must enrich neither the balance nor the human history.
	if _, err := store.ReceiveHeartbeat(ctx, device.ID, request, now.Add(10*time.Second)); err != nil {
		t.Fatal(err)
	}

	activity, err := store.LoadDeviceActivity(ctx, device.ID, "local-bonus-30m")
	if err != nil {
		t.Fatal(err)
	}
	if activity.Origin != "device" || activity.Status != "completed" || activity.Details["minutes"] != "30" {
		t.Fatalf("local bonus activity=%+v", activity)
	}
	for _, kind := range []string{"local_created", "synchronized", "confirmed"} {
		if step := findActivityStep(activity.Steps, kind); step == nil || step.Occurrences != 1 {
			t.Fatalf("activity step %q missing or duplicated: %+v", kind, activity.Steps)
		}
	}

	if hidden, err := store.DeleteCompletedDeviceActivities(ctx, device.ID); err != nil || hidden != 2 {
		t.Fatalf("hide local activity hidden=%d err=%v", hidden, err)
	}
	if _, err := store.ReceiveHeartbeat(ctx, device.ID, request, now.Add(15*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.LoadDeviceActivity(ctx, device.ID, "local-bonus-30m"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("retry made a hidden activity visible again: %v", err)
	}
	summary, err := store.LoadDailySummary(ctx, device.ID, "2026-08-10")
	if err != nil || summary.BonusSeconds != 30*60 {
		t.Fatalf("history cleanup changed the real bonus: summary=%+v err=%v", summary, err)
	}
}

func TestEveryRetainedAdministrativeDeviceMutationCreatesHumanActivity(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t)
	defer store.Close()
	now := time.Date(2026, time.August, 24, 12, 0, 0, 0, time.UTC)
	device, err := store.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.RenameDevice(ctx, device.ID, "Zorin de testes", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	var quotas [7]int64
	quotas[time.Monday] = 2 * 60 * 60
	if err := store.SaveQuotas(ctx, device.ID, quotas, 15, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	days := [7]bool{false, true, true, true, true, true, false}
	routineID, err := store.SaveRoutine(ctx, device.ID, Routine{
		Name: "Estudo", Days: days, Start: 18 * 3600, End: 19 * 3600, Enabled: true,
	}, now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SaveRoutine(ctx, device.ID, Routine{
		ID: routineID, Name: "Estudo noturno", Days: days, Start: 19 * 3600, End: 20 * 3600, Enabled: true,
	}, now.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteRoutine(ctx, device.ID, routineID, now.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.SetLocalPassword(ctx, device.ID, "safe-verifier", now.Add(6*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.IssueDeviceToken(ctx, device.ID, now.Add(7*time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeDeviceToken(ctx, device.ID, now.Add(8*time.Second)); err != nil {
		t.Fatal(err)
	}
	for index, command := range []string{"pause_monitoring", "resume_monitoring", "block_now", "clear_manual_block"} {
		if _, err := store.QueueControlOperation(ctx, device.ID, command, now.Add(time.Duration(9+index)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.QueueRemoteBonus(ctx, device.ID, 30*60, now.Add(13*time.Second)); err != nil {
		t.Fatal(err)
	}

	activities, err := store.ListDeviceActivities(ctx, device.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{
		"device_created": 1, "device_renamed": 1, "quotas_updated": 1,
		"routine_saved": 2, "routine_deleted": 1, "local_password_changed": 1,
		"device_token_issued": 1, "device_token_revoked": 1,
		"clear_manual_block": 1, "add_bonus": 1,
	}
	got := map[string]int{}
	dedicatedLifecycle := map[string]bool{
		"add_bonus": true, "clear_manual_block": true,
	}
	for _, activity := range activities {
		got[activity.Kind]++
		if !dedicatedLifecycle[activity.Kind] {
			completed := findActivityStep(activity.Steps, "completed")
			if activity.Status != "completed" || completed == nil || completed.Actor != "server" {
				t.Fatalf("server-side activity is not complete: %+v", activity)
			}
		}
	}
	for kind, count := range want {
		if got[kind] != count {
			t.Errorf("activity kind %q count=%d want=%d; all=%+v", kind, got[kind], count, got)
		}
	}
	if len(activities) != 11 {
		t.Fatalf("administrative activity total=%d want=11; kinds=%+v", len(activities), got)
	}
}

func findActivityStep(steps []ActivityStep, kind string) *ActivityStep {
	for index := range steps {
		if steps[index].Kind == kind {
			return &steps[index]
		}
	}
	return nil
}
