package alert

import (
	"testing"
	"time"

	"github.com/sergio/compasso/agent/policy"
)

func TestUpcomingAlertsRoutine(t *testing.T) {
	now := time.Date(2026, 8, 8, 21, 45, 0, 0, time.Local)
	blockAt := time.Date(2026, 8, 8, 22, 0, 0, 0, time.Local)
	decision := policy.Decision{Allowed: true, NextBlockAt: blockAt, NextBlockReason: policy.ReasonRoutine}
	alerts, err := UpcomingAlerts(decision, 10, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(alerts))
	}
	if alerts[0].At != time.Date(2026, 8, 8, 21, 50, 0, 0, time.Local) {
		t.Fatalf("expected first alert at 21:50, got %v", alerts[0].At)
	}
	if alerts[1].Kind != AlertFiveMinute || alerts[2].Kind != AlertOneMinute {
		t.Fatalf("unexpected alert kinds = %v, want [five one]", []string{alerts[1].Kind, alerts[2].Kind})
	}
}

func TestUpcomingAlertsQuota(t *testing.T) {
	now := time.Date(2026, 8, 8, 18, 45, 0, 0, time.Local)
	blockAt := time.Date(2026, 8, 8, 19, 0, 0, 0, time.Local)
	decision := policy.Decision{Allowed: true, NextBlockAt: blockAt, NextBlockReason: policy.ReasonQuota}
	alerts, err := UpcomingAlerts(decision, 10, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		time.Date(2026, 8, 8, 18, 50, 0, 0, time.Local),
		time.Date(2026, 8, 8, 18, 55, 0, 0, time.Local),
		time.Date(2026, 8, 8, 18, 59, 0, 0, time.Local),
	}
	if len(alerts) != len(want) {
		t.Fatalf("alerts=%d, want %d", len(alerts), len(want))
	}
	for index := range want {
		if !alerts[index].At.Equal(want[index]) {
			t.Fatalf("alert %d at %v, want %v", index, alerts[index].At, want[index])
		}
	}
}

func TestUpcomingAlertsAfterPause(t *testing.T) {
	now := time.Date(2026, 8, 8, 21, 50, 0, 0, time.Local)
	decision := policy.Decision{Allowed: true, Reason: policy.ReasonPaused}
	alerts, err := UpcomingAlerts(decision, 10, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts when paused, got %d", len(alerts))
	}
}

func TestUpcomingAlertsManualBlockNow(t *testing.T) {
	now := time.Now()
	decision := policy.Decision{Allowed: false, Reason: policy.ReasonManualBlock}
	alerts, err := UpcomingAlerts(decision, 10, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 0 {
		t.Fatalf("expected no alerts for manual block, got %d", len(alerts))
	}
}

func TestUpcomingAlertsSkipsPastPrimaryAlert(t *testing.T) {
	now := time.Date(2026, 8, 8, 21, 54, 0, 0, time.Local)
	blockAt := time.Date(2026, 8, 8, 22, 0, 0, 0, time.Local)
	decision := policy.Decision{Allowed: true, NextBlockAt: blockAt, NextBlockReason: policy.ReasonQuota}
	alerts, err := UpcomingAlerts(decision, 10, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts after missing primary, got %d", len(alerts))
	}
	if alerts[0].Kind != AlertFiveMinute || alerts[1].Kind != AlertOneMinute {
		t.Fatalf("unexpected alert kinds = %v", []string{alerts[0].Kind, alerts[1].Kind})
	}
}

func TestDueAlertsReturnsOnlyCrossedThreshold(t *testing.T) {
	blockAt := time.Date(2026, 8, 8, 14, 2, 0, 0, time.Local)
	decision := policy.Decision{Allowed: true, NextBlockAt: blockAt, NextBlockReason: policy.ReasonQuota}
	dueAlerts, err := DueAlerts(
		decision, 1,
		time.Date(2026, 8, 8, 14, 0, 59, 0, time.Local),
		time.Date(2026, 8, 8, 14, 1, 0, 0, time.Local),
	)
	if err != nil || len(dueAlerts) != 1 || dueAlerts[0].Kind != AlertOneMinute {
		t.Fatalf("due alerts=%+v err=%v", dueAlerts, err)
	}
	dueAlerts, err = DueAlerts(
		decision, 1,
		time.Date(2026, 8, 8, 14, 1, 0, 0, time.Local),
		time.Date(2026, 8, 8, 14, 1, 1, 0, time.Local),
	)
	if err != nil || len(dueAlerts) != 0 {
		t.Fatalf("threshold was delivered repeatedly: %+v err=%v", dueAlerts, err)
	}
}
