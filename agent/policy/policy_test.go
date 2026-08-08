package policy

import (
	"testing"
	"time"
)

var testLocation = time.FixedZone("America/Sao_Paulo", -3*60*60)

func instant(year int, month time.Month, day, hour, minute int) time.Time {
	return time.Date(year, month, day, hour, minute, 0, 0, testLocation)
}

func weekdays(days ...time.Weekday) (selected [7]bool) {
	for _, day := range days {
		selected[day] = true
	}
	return selected
}

func baseInput(now time.Time) Input {
	var quota WeeklyQuota
	for day := range quota {
		quota[day] = 2 * time.Hour
	}
	return Input{Now: now, Monitoring: MonitoringActive, Quota: quota}
}

func TestMondayQuotaReturnsThirtyMinutes(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 18, 0)) // Monday
	in.Consumed = 90 * time.Minute
	d := mustEvaluate(t, in)
	assertDecision(t, d, true, ReasonAllowed, true)
	if d.Remaining != 30*time.Minute {
		t.Fatalf("remaining = %s, want 30m", d.Remaining)
	}
	if want := in.Now.Add(30 * time.Minute); !d.NextBlockAt.Equal(want) || d.NextBlockReason != ReasonQuota {
		t.Fatalf("next block = %v/%s, want %v/%s", d.NextBlockAt, d.NextBlockReason, want, ReasonQuota)
	}
}

func TestBonusDoesNotOverrideRoutine(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 22, 30))
	in.Consumed, in.Bonus = 2*time.Hour, 30*time.Minute
	in.Routines = []Routine{{Name: "sleep", Days: weekdays(time.Monday), Start: 22 * time.Hour, End: 8 * time.Hour}}
	d := mustEvaluate(t, in)
	assertDecision(t, d, false, ReasonRoutine, false)
	if d.Remaining != 30*time.Minute || d.ActiveRoutine != "sleep" {
		t.Fatalf("unexpected routine decision: %+v", d)
	}
}

func TestOvernightRoutineBoundaries(t *testing.T) {
	routine := Routine{Name: "sleep", Days: weekdays(time.Monday), Start: 22 * time.Hour, End: 8 * time.Hour}
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"before start", instant(2026, time.August, 3, 21, 59), true},
		{"at start", instant(2026, time.August, 3, 22, 0), false},
		{"late", instant(2026, time.August, 3, 23, 30), false},
		{"next day", instant(2026, time.August, 4, 7, 59), false},
		{"at exclusive end", instant(2026, time.August, 4, 8, 0), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := baseInput(tt.now)
			in.Routines = []Routine{routine}
			d := mustEvaluate(t, in)
			if d.Allowed != tt.want {
				t.Fatalf("allowed = %v, want %v (decision %+v)", d.Allowed, tt.want, d)
			}
		})
	}
}

func TestPausedMonitoringOverridesEverything(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 23, 0))
	in.Monitoring, in.ManualBlock = MonitoringPaused, true
	in.Quota[in.Now.Weekday()] = 0
	in.Routines = []Routine{{Name: "sleep", Days: weekdays(time.Monday), Start: 22 * time.Hour, End: 8 * time.Hour}}
	d := mustEvaluate(t, in)
	assertDecision(t, d, true, ReasonPaused, false)
	if !d.NextBlockAt.IsZero() {
		t.Fatalf("paused decision must not schedule a block: %+v", d)
	}
}

func TestResumeDuringRoutineBlocksImmediately(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 23, 0))
	in.Routines = []Routine{{Name: "sleep", Days: weekdays(time.Monday), Start: 22 * time.Hour, End: 8 * time.Hour}}
	in.Monitoring = MonitoringPaused
	assertDecision(t, mustEvaluate(t, in), true, ReasonPaused, false)
	in.Monitoring = MonitoringActive
	assertDecision(t, mustEvaluate(t, in), false, ReasonRoutine, false)
}

func TestManualBlockPrecedesPositiveQuota(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 12, 0))
	in.ManualBlock = true
	assertDecision(t, mustEvaluate(t, in), false, ReasonManualBlock, false)
}

func TestAllWeekdaysAndExactBoundaries(t *testing.T) {
	// 2 August 2026 is Sunday, allowing a compact full-week matrix.
	for offset := 0; offset < 7; offset++ {
		day := time.Weekday(offset)
		routine := Routine{Name: day.String(), Days: weekdays(day), Start: 10 * time.Hour, End: 11 * time.Hour}
		date := instant(2026, time.August, 2+offset, 0, 0)
		for _, tc := range []struct {
			hour, minute int
			allowed      bool
		}{{9, 59, true}, {10, 0, false}, {10, 59, false}, {11, 0, true}} {
			in := baseInput(instant(date.Year(), date.Month(), date.Day(), tc.hour, tc.minute))
			in.Routines = []Routine{routine}
			if got := mustEvaluate(t, in).Allowed; got != tc.allowed {
				t.Errorf("%s %02d:%02d: allowed = %v, want %v", day, tc.hour, tc.minute, got, tc.allowed)
			}
		}
	}
}

func TestNextRoutineWinsWhenEarlierThanQuota(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 20, 0))
	in.Routines = []Routine{{Name: "sleep", Days: weekdays(time.Monday), Start: 22 * time.Hour, End: 8 * time.Hour}}
	d := mustEvaluate(t, in)
	if want := instant(2026, time.August, 3, 22, 0); !d.NextBlockAt.Equal(want) || d.NextBlockReason != ReasonRoutine {
		t.Fatalf("next block = %v/%s, want %v/%s", d.NextBlockAt, d.NextBlockReason, want, ReasonRoutine)
	}
}

func TestQuotaAndFullDayRoutine(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 0, 0))
	in.Consumed = 2 * time.Hour
	assertDecision(t, mustEvaluate(t, in), false, ReasonQuota, false)
	in.Consumed = 0
	in.Routines = []Routine{{Name: "all day", Days: weekdays(time.Monday)}}
	assertDecision(t, mustEvaluate(t, in), false, ReasonRoutine, false)
}

func TestRejectsInvalidInput(t *testing.T) {
	in := baseInput(instant(2026, time.August, 3, 12, 0))
	in.Bonus = -time.Second
	if _, err := Evaluate(in); err == nil {
		t.Fatal("expected negative bonus to be rejected")
	}
	in = baseInput(instant(2026, time.August, 3, 12, 0))
	in.Routines = []Routine{{Start: 24 * time.Hour}}
	if _, err := Evaluate(in); err == nil {
		t.Fatal("expected invalid routine time to be rejected")
	}
}

func mustEvaluate(t *testing.T, in Input) Decision {
	t.Helper()
	d, err := Evaluate(in)
	if err != nil {
		t.Fatalf("Evaluate() error: %v", err)
	}
	return d
}

func assertDecision(t *testing.T, got Decision, allowed bool, reason Reason, count bool) {
	t.Helper()
	if got.Allowed != allowed || got.Reason != reason || got.ShouldCount != count {
		t.Fatalf("decision = %+v, want allowed=%v reason=%s count=%v", got, allowed, reason, count)
	}
}
