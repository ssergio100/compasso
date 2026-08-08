package alert

import (
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
	if decision.NextBlockAt.IsZero() || !decision.Allowed {
		return nil, nil
	}
	if !decision.NextBlockAt.After(now) {
		return nil, nil
	}

	thresholds := map[time.Time]string{}
	add := func(at time.Time, kind string) {
		if !at.After(now) || !at.Before(decision.NextBlockAt) {
			return
		}
		thresholds[at] = kind
	}

	mainAlert := decision.NextBlockAt.Add(-time.Duration(warningMinutes) * time.Minute)
	add(mainAlert, AlertPrimary)
	add(decision.NextBlockAt.Add(-5*time.Minute), AlertFiveMinute)
	add(decision.NextBlockAt.Add(-1*time.Minute), AlertOneMinute)

	if len(thresholds) == 0 {
		return nil, nil
	}

	alertTimes := make([]time.Time, 0, len(thresholds))
	for at := range thresholds {
		alertTimes = append(alertTimes, at)
	}
	sort.Slice(alertTimes, func(i, j int) bool { return alertTimes[i].Before(alertTimes[j]) })

	alerts := make([]Alert, 0, len(alertTimes))
	for _, at := range alertTimes {
		kind := thresholds[at]
		alerts = append(alerts, Alert{
			At:     at,
			Kind:   kind,
			Reason: decision.NextBlockReason,
			Title:  titleForKind(kind, decision.NextBlockReason),
			Body:   bodyForKind(kind, at, decision.NextBlockAt, decision.NextBlockReason),
		})
	}
	return alerts, nil
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
