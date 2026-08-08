// Package policy contains the deterministic, infrastructure-independent policy
// engine used by tempo-agent.
package policy

import (
	"errors"
	"fmt"
	"time"
)

// MonitoringState controls whether restrictions and accounting are enabled.
type MonitoringState uint8

const (
	MonitoringActive MonitoringState = iota
	MonitoringPaused
)

// Reason explains why a decision was made.
type Reason string

const (
	ReasonAllowed     Reason = "allowed"
	ReasonPaused      Reason = "monitoring_paused"
	ReasonManualBlock Reason = "manual_block"
	ReasonRoutine     Reason = "routine"
	ReasonQuota       Reason = "quota_exhausted"
)

// WeeklyQuota stores the allowed duration for each weekday. time.Weekday is
// suitable for indexing it (Sunday == 0 through Saturday == 6).
type WeeklyQuota [7]time.Duration

// Routine describes a recurring blocked interval. Days identify the local day
// on which the routine starts. End is exclusive. Equal start and end denotes a
// full-day routine on each selected day.
type Routine struct {
	Name  string
	Days  [7]bool
	Start time.Duration // elapsed since local midnight
	End   time.Duration // elapsed since local midnight
}

// Input is a complete snapshot required to evaluate the current policy.
type Input struct {
	Now         time.Time
	Monitoring  MonitoringState
	ManualBlock bool
	Quota       WeeklyQuota
	Consumed    time.Duration
	Bonus       time.Duration
	Routines    []Routine
}

// Decision is the result of evaluating one immutable Input snapshot.
type Decision struct {
	Allowed         bool
	Reason          Reason
	Remaining       time.Duration
	ShouldCount     bool
	ActiveRoutine   string
	NextBlockAt     time.Time
	NextBlockReason Reason
}

// Evaluate applies the precedence rules and calculates the next predictable
// block. Durations are kept at their original precision (normally seconds).
func Evaluate(in Input) (Decision, error) {
	if err := validate(in); err != nil {
		return Decision{}, err
	}

	remaining := in.Quota[in.Now.Weekday()] + in.Bonus - in.Consumed
	if remaining < 0 {
		remaining = 0
	}
	d := Decision{Remaining: remaining}

	if in.Monitoring == MonitoringPaused {
		d.Allowed, d.Reason = true, ReasonPaused
		return d, nil
	}
	if in.ManualBlock {
		d.Reason = ReasonManualBlock
		return d, nil
	}
	if routine, ok := activeRoutine(in.Now, in.Routines); ok {
		d.Reason, d.ActiveRoutine = ReasonRoutine, routine.Name
		return d, nil
	}
	if remaining == 0 {
		d.Reason = ReasonQuota
		return d, nil
	}

	d.Allowed, d.Reason, d.ShouldCount = true, ReasonAllowed, true
	quotaBlock := in.Now.Add(remaining)
	routineBlock, hasRoutine := nextRoutineStart(in.Now, in.Routines)
	// A routine wins ties because it precedes quota exhaustion in the policy.
	if hasRoutine && !routineBlock.After(quotaBlock) {
		d.NextBlockAt, d.NextBlockReason = routineBlock, ReasonRoutine
	} else {
		d.NextBlockAt, d.NextBlockReason = quotaBlock, ReasonQuota
	}
	return d, nil
}

func validate(in Input) error {
	if in.Now.IsZero() {
		return errors.New("now must be set")
	}
	if in.Monitoring != MonitoringActive && in.Monitoring != MonitoringPaused {
		return errors.New("invalid monitoring state")
	}
	if in.Consumed < 0 || in.Bonus < 0 {
		return errors.New("consumed time and bonus cannot be negative")
	}
	for day, quota := range in.Quota {
		if quota < 0 {
			return fmt.Errorf("quota for weekday %d cannot be negative", day)
		}
	}
	for i, routine := range in.Routines {
		if routine.Start < 0 || routine.Start >= 24*time.Hour || routine.End < 0 || routine.End >= 24*time.Hour {
			return fmt.Errorf("routine %d has time outside the local day", i)
		}
	}
	return nil
}

func activeRoutine(now time.Time, routines []Routine) (Routine, bool) {
	for _, routine := range routines {
		startToday := atLocalTime(now, routine.Start)
		if routine.Start == routine.End {
			if routine.Days[now.Weekday()] {
				return routine, true
			}
			continue
		}
		if routine.Start < routine.End {
			if routine.Days[now.Weekday()] && !now.Before(startToday) && now.Before(atLocalTime(now, routine.End)) {
				return routine, true
			}
			continue
		}

		// Overnight intervals belong to their starting weekday. After midnight,
		// inspect the previous local day rather than the current one.
		if routine.Days[now.Weekday()] && !now.Before(startToday) {
			return routine, true
		}
		previousDay := now.AddDate(0, 0, -1).Weekday()
		if routine.Days[previousDay] && now.Before(atLocalTime(now, routine.End)) {
			return routine, true
		}
	}
	return Routine{}, false
}

func nextRoutineStart(now time.Time, routines []Routine) (time.Time, bool) {
	var next time.Time
	for offset := 0; offset <= 7; offset++ {
		day := now.AddDate(0, 0, offset)
		for _, routine := range routines {
			if !routine.Days[day.Weekday()] {
				continue
			}
			candidate := atLocalTime(day, routine.Start)
			if !candidate.After(now) {
				continue
			}
			if next.IsZero() || candidate.Before(next) {
				next = candidate
			}
		}
	}
	return next, !next.IsZero()
}

func atLocalTime(day time.Time, sinceMidnight time.Duration) time.Time {
	year, month, date := day.Date()
	hour := int(sinceMidnight / time.Hour)
	sinceMidnight %= time.Hour
	minute := int(sinceMidnight / time.Minute)
	sinceMidnight %= time.Minute
	second := int(sinceMidnight / time.Second)
	nanosecond := int(sinceMidnight % time.Second)
	return time.Date(year, month, date, hour, minute, second, nanosecond, day.Location())
}
