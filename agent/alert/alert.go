package alert

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/sergio/compasso/agent/policy"
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

// UpcomingAlerts returns the remaining pre-block alerts for the current decision.
// It returns an empty slice when the monitoring state does not produce a future block,
// when warningMinutes is negative, or when the next block is already active.
func UpcomingAlerts(decision policy.Decision, warningMinutes int, now time.Time) ([]Alert, error) {
	if warningMinutes < 0 {
		return nil, fmt.Errorf("warning minutes cannot be negative")
	}
	if now.IsZero() {
		return nil, fmt.Errorf("now must be set")
	}
	plannedAlerts := plannedAlerts(decision, warningMinutes)
	upcomingAlerts := make([]Alert, 0, len(plannedAlerts))
	for _, plannedAlert := range plannedAlerts {
		if plannedAlert.At.After(now) {
			upcomingAlerts = append(upcomingAlerts, plannedAlert)
		}
	}
	return upcomingAlerts, nil
}

// DueAlerts returns notification thresholds crossed since the previous daemon
// cycle. A zero previousCycle means startup and deliberately emits nothing.
func DueAlerts(decision policy.Decision, warningMinutes int, previousCycle, now time.Time) ([]Alert, error) {
	if warningMinutes < 0 {
		return nil, fmt.Errorf("warning minutes cannot be negative")
	}
	if now.IsZero() || (!previousCycle.IsZero() && now.Before(previousCycle)) {
		return nil, fmt.Errorf("alert cycle times are invalid")
	}
	if previousCycle.IsZero() {
		return nil, nil
	}
	var dueAlerts []Alert
	for _, plannedAlert := range plannedAlerts(decision, warningMinutes) {
		if plannedAlert.At.After(previousCycle) && !plannedAlert.At.After(now) {
			dueAlerts = append(dueAlerts, plannedAlert)
		}
	}
	return dueAlerts, nil
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
