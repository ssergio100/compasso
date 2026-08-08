// Package pamgate implements the small access check used from the PAM login
// stack. It reads only durable local state and never requires network access.
package pamgate

import (
	"context"
	"fmt"
	"time"

	"github.com/sergio/compasso/agent/policy"
	"github.com/sergio/compasso/agent/storage"
)

// Result is the access decision returned to the PAM helper.
type Result struct {
	Allowed bool
	Reason  policy.Reason
}

// Check evaluates whether username may start a new graphical session. Accounts
// other than controlledUser are always outside Compasso's policy.
func Check(ctx context.Context, store *storage.Store, controlledUser, username string, now time.Time) (Result, error) {
	if store == nil {
		return Result{}, fmt.Errorf("store is required")
	}
	if controlledUser == "" || username == "" {
		return Result{}, fmt.Errorf("controlled user and PAM user are required")
	}
	if username != controlledUser {
		return Result{Allowed: true, Reason: policy.ReasonAllowed}, nil
	}
	if now.IsZero() {
		return Result{}, fmt.Errorf("current time is required")
	}

	snapshot, err := store.LoadPolicy(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("load local policy: %w", err)
	}
	localDate := now.Format("2006-01-02")
	usage, err := store.LoadDailyUsage(ctx, localDate)
	if err != nil {
		return Result{}, fmt.Errorf("load daily usage: %w", err)
	}
	bonusSeconds, err := store.TotalBonusSeconds(ctx, localDate)
	if err != nil {
		return Result{}, fmt.Errorf("load daily bonus: %w", err)
	}

	monitoring := policy.MonitoringActive
	if snapshot.MonitoringPaused {
		monitoring = policy.MonitoringPaused
	}
	input := policy.Input{
		Now:         now,
		Monitoring:  monitoring,
		ManualBlock: snapshot.ManualBlock,
		Quota:       policy.WeeklyQuota(snapshot.WeeklyQuota),
		Consumed:    time.Duration(usage.SecondsUsed) * time.Second,
		Bonus:       time.Duration(bonusSeconds) * time.Second,
	}
	for _, routine := range snapshot.Routines {
		if !routine.Enabled {
			continue
		}
		input.Routines = append(input.Routines, policy.Routine{
			Name: routine.Name, Days: routine.Days, Start: routine.Start, End: routine.End,
		})
	}
	decision, err := policy.Evaluate(input)
	if err != nil {
		return Result{}, fmt.Errorf("evaluate local policy: %w", err)
	}
	return Result{Allowed: decision.Allowed, Reason: decision.Reason}, nil
}
