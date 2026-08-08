package policy

import (
	"testing"
	"time"
)

func TestAcceleratedWeekPreservesDailyQuotasAndOvernightRoutine(t *testing.T) {
	location := time.FixedZone("America/Sao_Paulo", -3*60*60)
	weekStart := time.Date(2026, time.August, 9, 0, 0, 0, 0, location) // Sunday.
	var weeklyQuota WeeklyQuota
	for weekday := time.Sunday; weekday <= time.Saturday; weekday++ {
		weeklyQuota[weekday] = time.Duration(int(weekday)+1) * time.Hour
	}
	var weekdayRoutineDays [7]bool
	for weekday := time.Monday; weekday <= time.Friday; weekday++ {
		weekdayRoutineDays[weekday] = true
	}
	sleepRoutine := Routine{
		Name: "Dormir", Days: weekdayRoutineDays,
		Start: 22 * time.Hour, End: 8 * time.Hour,
	}

	for dayOffset := 0; dayOffset < 7; dayOffset++ {
		currentDay := weekStart.AddDate(0, 0, dayOffset)
		weekday := currentDay.Weekday()
		midday := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), 12, 0, 0, 0, location)
		decision, err := Evaluate(Input{
			Now: midday, Monitoring: MonitoringActive, Quota: weeklyQuota,
			Consumed: 30 * time.Minute, Routines: []Routine{sleepRoutine},
		})
		if err != nil || !decision.Allowed {
			t.Fatalf("%s midday decision=%+v err=%v", weekday, decision, err)
		}
		expectedRemaining := weeklyQuota[weekday] - 30*time.Minute
		if decision.Remaining != expectedRemaining {
			t.Fatalf("%s remaining=%s want=%s", weekday, decision.Remaining, expectedRemaining)
		}

		lateEvening := time.Date(currentDay.Year(), currentDay.Month(), currentDay.Day(), 22, 30, 0, 0, location)
		decision, err = Evaluate(Input{
			Now: lateEvening, Monitoring: MonitoringActive, Quota: weeklyQuota,
			Routines: []Routine{sleepRoutine},
		})
		shouldBeBlocked := weekday >= time.Monday && weekday <= time.Friday
		if err != nil || decision.Allowed == shouldBeBlocked {
			t.Fatalf("%s overnight routine decision=%+v err=%v", weekday, decision, err)
		}
		if shouldBeBlocked && decision.Reason != ReasonRoutine {
			t.Fatalf("%s block reason=%s want=%s", weekday, decision.Reason, ReasonRoutine)
		}
	}

	pausedDecision, err := Evaluate(Input{
		Now:        weekStart.AddDate(0, 0, 1).Add(23 * time.Hour),
		Monitoring: MonitoringPaused, Quota: weeklyQuota, Consumed: 24 * time.Hour,
		Routines: []Routine{sleepRoutine},
	})
	if err != nil || !pausedDecision.Allowed || pausedDecision.ShouldCount {
		t.Fatalf("paused accelerated week decision=%+v err=%v", pausedDecision, err)
	}
}
