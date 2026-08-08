// Package daemon coordinates policy evaluation, accounting and enforcement.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sergio/compasso/agent/policy"
	"github.com/sergio/compasso/agent/session"
	"github.com/sergio/compasso/agent/storage"
)

const localDateLayout = "2006-01-02"

// Status describes one completed daemon cycle.
type Status struct {
	Decision         policy.Decision
	GraphicalSession bool
	UsageSeconds     int64
}

// Daemon owns the runtime state that is intentionally not persisted between
// individual polling cycles. Durable policy and usage remain in Store.
type Daemon struct {
	store           *storage.Store
	sessions        session.Manager
	controlledUser  string
	checkpointEvery time.Duration

	tracker         *storage.UsageTracker
	trackerDate     string
	lastAt          time.Time
	lastShouldCount bool
	lastHadSession  bool
	lastCountUntil  time.Time
	fraction        time.Duration
	terminated      map[string]bool
}

// New creates a daemon controller. Call Step periodically or Run for the
// production loop.
func New(store *storage.Store, sessions session.Manager, controlledUser string, checkpointEvery time.Duration) (*Daemon, error) {
	if store == nil || sessions == nil {
		return nil, errors.New("store and session manager are required")
	}
	if controlledUser == "" {
		return nil, errors.New("controlled user cannot be empty")
	}
	if checkpointEvery <= 0 {
		return nil, errors.New("checkpoint interval must be positive")
	}
	return &Daemon{
		store: store, sessions: sessions, controlledUser: controlledUser,
		checkpointEvery: checkpointEvery, terminated: make(map[string]bool),
	}, nil
}

// Step performs one complete observation, accounting and enforcement cycle.
// now should contain Go's monotonic component in production.
func (d *Daemon) Step(ctx context.Context, now time.Time) (Status, error) {
	if now.IsZero() {
		return Status{}, errors.New("step time must be set")
	}
	if !d.lastAt.IsZero() && now.Before(d.lastAt) {
		return Status{}, errors.New("step time moved backwards")
	}

	snapshot, err := d.store.LoadPolicy(ctx)
	if err != nil {
		return Status{}, err
	}
	allSessions, err := d.sessions.Sessions(ctx, d.controlledUser)
	if err != nil {
		return Status{}, err
	}
	graphical := graphicalSessions(allSessions)
	localDate := now.Format(localDateLayout)
	if d.tracker == nil {
		if err := d.prepareTracker(ctx, localDate, now); err != nil {
			return Status{}, err
		}
	} else if d.trackerDate != localDate {
		midnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if d.lastShouldCount && d.lastHadSession {
			beforeMidnight := cappedElapsed(d.lastAt, midnight, d.lastCountUntil)
			if err := d.addElapsed(ctx, beforeMidnight, midnight); err != nil {
				return Status{}, fmt.Errorf("account usage before local midnight: %w", err)
			}
		}
		if err := d.prepareTracker(ctx, localDate, now); err != nil {
			return Status{}, err
		}
		newDayDecision, err := d.evaluate(ctx, snapshot, midnight)
		if err != nil {
			return Status{}, err
		}
		if newDayDecision.ShouldCount && len(graphical) != 0 {
			afterMidnight := cappedElapsed(midnight, now, newDayDecision.NextBlockAt)
			if err := d.addElapsed(ctx, afterMidnight, now); err != nil {
				return Status{}, fmt.Errorf("account usage after local midnight: %w", err)
			}
		}
	} else if !d.lastAt.IsZero() && d.lastShouldCount && d.lastHadSession {
		elapsed := cappedElapsed(d.lastAt, now, d.lastCountUntil)
		if err := d.addElapsed(ctx, elapsed, now); err != nil {
			return Status{}, fmt.Errorf("account allowed usage: %w", err)
		}
	}

	decision, err := d.evaluate(ctx, snapshot, now)
	if err != nil {
		return Status{}, err
	}
	if !decision.Allowed && d.lastShouldCount {
		if err := d.tracker.Flush(ctx, now); err != nil {
			return Status{}, fmt.Errorf("persist usage before blocking login: %w", err)
		}
	}
	status := Status{
		Decision: decision, GraphicalSession: len(graphical) != 0,
		UsageSeconds: d.tracker.Seconds(),
	}
	// Advance runtime accounting state before enforcement. A transient logout
	// failure must not cause the same elapsed interval to be counted twice.
	d.lastAt = now
	d.lastShouldCount = decision.ShouldCount
	d.lastHadSession = len(graphical) != 0
	d.lastCountUntil = decision.NextBlockAt

	if decision.Allowed {
		for sessionID := range d.terminated {
			delete(d.terminated, sessionID)
		}
	} else {
		for _, current := range graphical {
			if d.terminated[current.ID] {
				continue
			}
			if err := d.sessions.Terminate(ctx, current.ID); err != nil {
				return status, err
			}
			d.terminated[current.ID] = true
		}
	}
	return status, nil
}

// Flush persists the final in-memory counter during an orderly shutdown.
func (d *Daemon) Flush(ctx context.Context, now time.Time) error {
	if d.tracker == nil {
		return nil
	}
	return d.tracker.Flush(ctx, now)
}

// Run executes Step immediately and then at every tick until cancellation.
// A missing initial policy is non-fatal: the daemon stays alive so a future
// synchronization phase can install one.
func (d *Daemon) Run(ctx context.Context, tick time.Duration, logger *log.Logger) error {
	if tick <= 0 {
		return errors.New("tick interval must be positive")
	}
	if logger == nil {
		logger = log.Default()
	}
	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	lastSummary := ""
	lastError := ""
	runStep := func(now time.Time) {
		status, err := d.Step(ctx, now)
		if err != nil {
			if err.Error() != lastError {
				logger.Printf("agent cycle failed: %v", err)
				lastError = err.Error()
			}
			return
		}
		if lastError != "" {
			logger.Printf("agent cycle recovered")
			lastError = ""
		}
		summary := fmt.Sprintf("%s:%t", status.Decision.Reason, status.GraphicalSession)
		if summary != lastSummary {
			logger.Printf("decision=%s session=%t usage_seconds=%d remaining_seconds=%d",
				status.Decision.Reason, status.GraphicalSession, status.UsageSeconds,
				int64(status.Decision.Remaining/time.Second))
			lastSummary = summary
		}
	}
	runStep(time.Now())
	for {
		select {
		case <-ctx.Done():
			flushContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := d.Flush(flushContext, time.Now())
			cancel()
			return err
		case now := <-ticker.C:
			runStep(now)
		}
	}
}

func (d *Daemon) prepareTracker(ctx context.Context, localDate string, now time.Time) error {
	if d.trackerDate == localDate {
		return nil
	}
	if d.tracker != nil {
		if err := d.tracker.Flush(ctx, now); err != nil {
			return fmt.Errorf("flush previous local date: %w", err)
		}
	}
	tracker, err := storage.NewUsageTracker(ctx, d.store, localDate, d.checkpointEvery)
	if err != nil {
		return fmt.Errorf("load usage for %s: %w", localDate, err)
	}
	d.tracker = tracker
	d.trackerDate = localDate
	d.fraction = 0
	return nil
}

func (d *Daemon) addElapsed(ctx context.Context, elapsed time.Duration, now time.Time) error {
	if elapsed <= 0 {
		return nil
	}
	total := d.fraction + elapsed
	whole := total.Truncate(time.Second)
	d.fraction = total - whole
	if whole == 0 {
		return nil
	}
	return d.tracker.Add(ctx, whole, now)
}

func (d *Daemon) evaluate(ctx context.Context, snapshot storage.PolicySnapshot, now time.Time) (policy.Decision, error) {
	bonus, err := d.store.TotalBonusSeconds(ctx, d.trackerDate)
	if err != nil {
		return policy.Decision{}, fmt.Errorf("load daily bonus: %w", err)
	}
	monitoring := policy.MonitoringActive
	if snapshot.MonitoringPaused {
		monitoring = policy.MonitoringPaused
	}
	input := policy.Input{
		Now: now, Monitoring: monitoring, ManualBlock: snapshot.ManualBlock,
		Quota:    policy.WeeklyQuota(snapshot.WeeklyQuota),
		Consumed: time.Duration(d.tracker.Seconds()) * time.Second,
		Bonus:    time.Duration(bonus) * time.Second,
	}
	for _, routine := range snapshot.Routines {
		if routine.Enabled {
			input.Routines = append(input.Routines, policy.Routine{
				Name: routine.Name, Days: routine.Days, Start: routine.Start, End: routine.End,
			})
		}
	}
	decision, err := policy.Evaluate(input)
	if err != nil {
		return policy.Decision{}, fmt.Errorf("evaluate policy: %w", err)
	}
	return decision, nil
}

func graphicalSessions(sessions []session.Session) []session.Session {
	graphical := make([]session.Session, 0, len(sessions))
	for _, current := range sessions {
		if current.IsLocalGraphical() {
			graphical = append(graphical, current)
		}
	}
	return graphical
}

func cappedElapsed(start, end, countUntil time.Time) time.Duration {
	if !countUntil.IsZero() && countUntil.Before(end) {
		end = countUntil
	}
	if !end.After(start) {
		return 0
	}
	return end.Sub(start)
}
