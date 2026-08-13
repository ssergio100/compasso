package alert

import (
	"testing"
	"time"

	"github.com/ssergio100/compasso/agent/policy"
)

func TestDueAlertsReturnsCrossedRoutineThresholds(t *testing.T) {
	blockAt := time.Date(2026, 8, 8, 22, 0, 0, 0, time.Local)
	decision := policy.Decision{Allowed: true, NextBlockAt: blockAt, NextBlockReason: policy.ReasonRoutine}
	alerts, err := DueAlerts(decision, 10,
		time.Date(2026, 8, 8, 21, 49, 59, 0, time.Local),
		time.Date(2026, 8, 8, 21, 59, 0, 0, time.Local))
	if err != nil || len(alerts) != 3 {
		t.Fatalf("routine alerts=%+v err=%v", alerts, err)
	}
	if alerts[0].Kind != AlertPrimary || alerts[1].Kind != AlertFiveMinute || alerts[2].Kind != AlertOneMinute {
		t.Fatalf("unexpected alert kinds: %+v", alerts)
	}
}

func TestDueAlertsReturnsNoAlertWithoutFutureBlock(t *testing.T) {
	now := time.Date(2026, 8, 8, 21, 50, 0, 0, time.Local)
	for _, decision := range []policy.Decision{
		{Allowed: true, Reason: policy.ReasonPaused},
		{Allowed: false, Reason: policy.ReasonManualBlock},
	} {
		alerts, err := DueAlerts(decision, 10, now.Add(-time.Second), now)
		if err != nil || len(alerts) != 0 {
			t.Fatalf("decision=%+v alerts=%+v err=%v", decision, alerts, err)
		}
	}
}

func TestDueAlertsReturnsOnlyNewlyCrossedThreshold(t *testing.T) {
	blockAt := time.Date(2026, 8, 8, 14, 2, 0, 0, time.Local)
	decision := policy.Decision{Allowed: true, NextBlockAt: blockAt, NextBlockReason: policy.ReasonQuota}
	dueAlerts, err := DueAlerts(decision, 1,
		time.Date(2026, 8, 8, 14, 0, 59, 0, time.Local),
		time.Date(2026, 8, 8, 14, 1, 0, 0, time.Local))
	if err != nil || len(dueAlerts) != 1 || dueAlerts[0].Kind != AlertOneMinute {
		t.Fatalf("due alerts=%+v err=%v", dueAlerts, err)
	}
	dueAlerts, err = DueAlerts(decision, 1,
		time.Date(2026, 8, 8, 14, 1, 0, 0, time.Local),
		time.Date(2026, 8, 8, 14, 1, 1, 0, time.Local))
	if err != nil || len(dueAlerts) != 0 {
		t.Fatalf("threshold was delivered repeatedly: %+v err=%v", dueAlerts, err)
	}
}

func TestDueAlertsRejectsInvalidInputs(t *testing.T) {
	decision := policy.Decision{Allowed: true, NextBlockAt: time.Now().Add(time.Hour)}
	if _, err := DueAlerts(decision, -1, time.Now(), time.Now()); err == nil {
		t.Fatal("negative warning unexpectedly accepted")
	}
	if _, err := DueAlerts(decision, 10, time.Now(), time.Time{}); err == nil {
		t.Fatal("zero current time unexpectedly accepted")
	}
}
