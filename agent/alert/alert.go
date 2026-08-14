package alert

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/ssergio100/compasso/agent/policy"
)

const (
	AlertPrimary    = "primary"
	AlertFiveMinute = "five_minute"
	AlertOneMinute  = "one_minute"
)

// Alert represents a planned notification event before a known block.
type Alert struct {
	At     time.Time
	Kind   string
	Reason policy.Reason
	Title  string
	Body   string
}

type Notifier interface {
	Notify(context.Context, Alert) error
}

// Cycle identifies one authorization period for alert deduplication.
type Cycle struct {
	PolicyRevision int64
	SessionID      string
	LocalDate      string
}

// Tracker emits each threshold once per authorization period. Its zero value
// is ready for use and deliberately emits nothing on the first observation.
type Tracker struct {
	initialized       bool
	active            bool
	cycle             Cycle
	reason            policy.Reason
	previousCycle     time.Time
	previousRemaining time.Duration
	emitted           map[string]bool
}

// Due returns thresholds crossed since the previous observation. When deliver
// is false, crossed thresholds are consumed without being returned so a locked
// desktop cannot accumulate obsolete notifications.
func (tracker *Tracker) Due(decision policy.Decision, warningMinutes int, cycle Cycle, now time.Time, deliver bool) ([]Alert, error) {
	if warningMinutes < 0 {
		return nil, fmt.Errorf("warning minutes cannot be negative")
	}
	if now.IsZero() || (tracker.initialized && now.Before(tracker.previousCycle)) {
		return nil, fmt.Errorf("alert cycle times are invalid")
	}
	if !tracker.initialized {
		tracker.initialized = true
		tracker.observe(decision, cycle, now)
		return nil, nil
	}

	hasFutureBlock := decision.Allowed && !decision.NextBlockAt.IsZero()
	newCycle := !tracker.active || tracker.cycle != cycle || tracker.reason != decision.NextBlockReason ||
		decision.Remaining > tracker.previousRemaining
	if !hasFutureBlock {
		tracker.active = false
		tracker.previousCycle = now
		tracker.previousRemaining = decision.Remaining
		return nil, nil
	}
	if newCycle {
		tracker.emitted = make(map[string]bool)
		tracker.observe(decision, cycle, now)
		return nil, nil
	}

	var dueAlerts []Alert
	for _, plannedAlert := range plannedAlerts(decision, warningMinutes) {
		crossed := plannedAlert.At.After(tracker.previousCycle) && !plannedAlert.At.After(now)
		if crossed && !tracker.emitted[plannedAlert.Kind] {
			tracker.emitted[plannedAlert.Kind] = true
			if !deliver {
				continue
			}
			dueAlerts = append(dueAlerts, plannedAlert)
		}
	}
	tracker.observe(decision, cycle, now)
	return dueAlerts, nil
}

func (tracker *Tracker) observe(decision policy.Decision, cycle Cycle, now time.Time) {
	tracker.active = decision.Allowed && !decision.NextBlockAt.IsZero()
	tracker.cycle = cycle
	tracker.reason = decision.NextBlockReason
	tracker.previousCycle = now
	tracker.previousRemaining = decision.Remaining
	if tracker.emitted == nil {
		tracker.emitted = make(map[string]bool)
	}
}

func plannedAlerts(decision policy.Decision, warningMinutes int) []Alert {
	if decision.NextBlockAt.IsZero() || !decision.Allowed {
		return nil
	}
	thresholds := map[time.Time]string{}
	addThreshold := func(alertTime time.Time, alertKind string) {
		if alertTime.Before(decision.NextBlockAt) {
			thresholds[alertTime] = alertKind
		}
	}
	addThreshold(decision.NextBlockAt.Add(-time.Duration(warningMinutes)*time.Minute), AlertPrimary)
	addThreshold(decision.NextBlockAt.Add(-5*time.Minute), AlertFiveMinute)
	addThreshold(decision.NextBlockAt.Add(-time.Minute), AlertOneMinute)

	alertTimes := make([]time.Time, 0, len(thresholds))
	for alertTime := range thresholds {
		alertTimes = append(alertTimes, alertTime)
	}
	sort.Slice(alertTimes, func(first, second int) bool { return alertTimes[first].Before(alertTimes[second]) })
	plannedAlerts := make([]Alert, 0, len(alertTimes))
	for _, alertTime := range alertTimes {
		alertKind := thresholds[alertTime]
		plannedAlerts = append(plannedAlerts, Alert{
			At: alertTime, Kind: alertKind, Reason: decision.NextBlockReason,
			Title: titleForKind(alertKind, decision.NextBlockReason),
			Body:  bodyForKind(alertKind, alertTime, decision.NextBlockAt, decision.NextBlockReason),
		})
	}
	return plannedAlerts
}

func titleForKind(kind string, reason policy.Reason) string {
	switch kind {
	case AlertPrimary:
		return "Aviso de bloqueio previsto"
	case AlertFiveMinute:
		return "Bloqueio em 5 minutos"
	case AlertOneMinute:
		return "Bloqueio em 1 minuto"
	default:
		return "Aviso de bloqueio"
	}
}

func bodyForKind(kind string, at, blockAt time.Time, reason policy.Reason) string {
	remaining := int(blockAt.Sub(at).Minutes())
	reasonText := ""
	if reason == policy.ReasonRoutine {
		reasonText = "rotina programada"
	} else if reason == policy.ReasonQuota {
		reasonText = "fim de cota"
	} else {
		reasonText = string(reason)
	}
	return fmt.Sprintf("O bloqueio por %s ocorrerá em %d minuto(s).", reasonText, remaining)
}
