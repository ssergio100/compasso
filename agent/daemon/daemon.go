// Package daemon coordinates policy evaluation, accounting and enforcement.
package daemon

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/sergio/compasso/agent/alert"
	"github.com/sergio/compasso/agent/policy"
	"github.com/sergio/compasso/agent/session"
	"github.com/sergio/compasso/agent/storage"
)

const (
	localDateLayout             = "2006-01-02"
	blockedReloginStabilization = 10 * time.Second
)

// Status describes one completed daemon cycle.
type Status struct {
	Decision                policy.Decision
	GraphicalSession        bool
	AwaitingSynchronization bool
	UsageSeconds            int64
	DueAlerts               []alert.Alert
}

// SynchronizationSource receives session presence for the next heartbeat.
type SynchronizationSource interface {
	SetGraphicalSession(active bool, sessionID string)
}

// Daemon owns the runtime state that is intentionally not persisted between
// individual polling cycles. Durable policy and usage remain in Store.
type Daemon struct {
	store           *storage.Store
	sessions        session.Manager
	controlledUser  string
	checkpointEvery time.Duration

	tracker               *storage.UsageTracker
	trackerDate           string
	lastAt                time.Time
	lastShouldCount       bool
	lastHadSession        bool
	lastCountUntil        time.Time
	fraction              time.Duration
	terminated            map[string]bool
	allowedSessions       map[string]bool
	blockedActiveAt       map[string]time.Time
	synchronizationSource SynchronizationSource
	alertNotifier         alert.Notifier
}

func (d *Daemon) SetAlertNotifier(notifier alert.Notifier) {
	d.alertNotifier = notifier
}

// SetSynchronizationSource enables server-confirmed balance authorization and
// graphical-session reporting.
func (d *Daemon) SetSynchronizationSource(source SynchronizationSource) {
	d.synchronizationSource = source
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
		checkpointEvery: checkpointEvery,
		terminated:      make(map[string]bool),
		allowedSessions: make(map[string]bool),
		blockedActiveAt: make(map[string]time.Time),
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

	snapshot, err := d.store.CurrentPolicy()
	if err != nil {
		return Status{}, err
	}
	allSessions, err := d.sessions.Sessions(ctx, d.controlledUser)
	if err != nil {
		return Status{}, err
	}
	graphical := graphicalSessions(allSessions)
	activeGraphicalSessionID := establishedGraphicalSessionID(graphical)
	if d.synchronizationSource != nil {
		d.synchronizationSource.SetGraphicalSession(activeGraphicalSessionID != "", activeGraphicalSessionID)
	}
	d.forgetSessionsNoLongerPresent(graphical)
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

	confirmedState, hasConfirmedState := d.store.CurrentConfirmedSessionState()
	if hasConfirmedState && confirmedState.LocalDate == localDate && d.tracker.Seconds() < confirmedState.UsageSeconds {
		if err := d.tracker.EnsureAtLeast(ctx, confirmedState.UsageSeconds, now); err != nil {
			return Status{}, fmt.Errorf("reconcile server-confirmed usage: %w", err)
		}
	}
	awaitingSynchronization := d.synchronizationSource != nil && activeGraphicalSessionID != "" &&
		(!hasConfirmedState || confirmedState.SessionID != activeGraphicalSessionID ||
			confirmedState.LocalDate != localDate || confirmedState.Revision < snapshot.Revision)
	var decision policy.Decision
	if awaitingSynchronization {
		decision = policy.Decision{
			Allowed: true, Reason: policy.ReasonAwaitingSynchronization,
		}
	} else {
		decision, err = d.evaluate(ctx, snapshot, now)
		if err != nil {
			return Status{}, err
		}
	}
	if !decision.Allowed && d.lastShouldCount {
		if err := d.tracker.Flush(ctx, now); err != nil {
			return Status{}, fmt.Errorf("persist usage before blocking login: %w", err)
		}
	}
	status := Status{
		Decision: decision, GraphicalSession: len(graphical) != 0,
		AwaitingSynchronization: awaitingSynchronization,
		UsageSeconds:            d.tracker.Seconds(),
	}
	if len(graphical) != 0 && !awaitingSynchronization {
		status.DueAlerts, err = alert.DueAlerts(decision, snapshot.WarningMinutes, d.lastAt, now)
		if err != nil {
			return Status{}, fmt.Errorf("calculate due alerts: %w", err)
		}
	}
	// Advance runtime accounting state before enforcement. A transient logout
	// failure must not cause the same elapsed interval to be counted twice.
	d.lastAt = now
	d.lastShouldCount = decision.ShouldCount
	d.lastHadSession = len(graphical) != 0
	d.lastCountUntil = decision.NextBlockAt
	if awaitingSynchronization {
		for _, current := range graphical {
			if current.State == "active" && !d.allowedSessions[current.ID] {
				if _, observed := d.blockedActiveAt[current.ID]; !observed {
					d.blockedActiveAt[current.ID] = now
				}
			}
		}
		return status, nil
	}

	if decision.Allowed {
		for sessionID := range d.terminated {
			delete(d.terminated, sessionID)
		}
		for _, current := range graphical {
			if current.State == "active" {
				d.allowedSessions[current.ID] = true
			}
			delete(d.blockedActiveAt, current.ID)
		}
	} else {
		for _, current := range graphical {
			if d.terminated[current.ID] {
				continue
			}
			if !d.allowedSessions[current.ID] {
				if current.State != "active" {
					continue
				}
				activeSince, observed := d.blockedActiveAt[current.ID]
				if !observed {
					d.blockedActiveAt[current.ID] = now
				}
				if !observed {
					continue
				}
				if now.Sub(activeSince) < blockedReloginStabilization {
					continue
				}
			} else if current.State != "active" {
				continue
			}
			if err := d.sessions.Logout(ctx, current); err != nil {
				return status, err
			}
			d.terminated[current.ID] = true
		}
	}
	return status, nil
}

// forgetSessionsNoLongerPresent prevents a reused logind session ID from
// inheriting the enforcement state of an older session.
func (d *Daemon) forgetSessionsNoLongerPresent(graphical []session.Session) {
	present := make(map[string]bool, len(graphical))
	for _, current := range graphical {
		present[current.ID] = true
	}
	for sessionID := range d.terminated {
		if !present[sessionID] {
			delete(d.terminated, sessionID)
		}
	}
	for sessionID := range d.allowedSessions {
		if !present[sessionID] {
			delete(d.allowedSessions, sessionID)
		}
	}
	for sessionID := range d.blockedActiveAt {
		if !present[sessionID] {
			delete(d.blockedActiveAt, sessionID)
		}
	}
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
		for _, dueAlert := range status.DueAlerts {
			if d.alertNotifier == nil {
				logger.Printf("desktop alert unavailable kind=%s", dueAlert.Kind)
				continue
			}
			if err := d.alertNotifier.Notify(ctx, dueAlert); err != nil {
				logger.Printf("desktop alert failed kind=%s: %v", dueAlert.Kind, err)
			}
		}
		summary := fmt.Sprintf("%s:%t:%t", status.Decision.Reason, status.GraphicalSession,
			status.AwaitingSynchronization)
		if summary != lastSummary {
			logger.Printf("decision=%s session=%t awaiting_synchronization=%t usage_seconds=%d remaining_seconds=%d",
				status.Decision.Reason, status.GraphicalSession, status.AwaitingSynchronization,
				status.UsageSeconds, int64(status.Decision.Remaining/time.Second))
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
	monitoring := policy.MonitoringActive
	if snapshot.MonitoringPaused {
		monitoring = policy.MonitoringPaused
	}
	input := policy.Input{
		Now: now, Monitoring: monitoring, ManualBlock: snapshot.ManualBlock,
	}
	if d.synchronizationSource == nil {
		bonus, err := d.store.TotalBonusSeconds(ctx, d.trackerDate)
		if err != nil {
			return policy.Decision{}, fmt.Errorf("load daily bonus: %w", err)
		}
		input.Quota = policy.WeeklyQuota(snapshot.WeeklyQuota)
		input.Consumed = time.Duration(d.tracker.Seconds()) * time.Second
		input.Bonus = time.Duration(bonus) * time.Second
	} else {
		confirmedState, available := d.store.CurrentConfirmedSessionState()
		remainingSeconds := int64(0)
		if available && confirmedState.LocalDate == d.trackerDate {
			usageSinceConfirmation := d.tracker.Seconds() - confirmedState.UsageSeconds
			if usageSinceConfirmation < 0 {
				usageSinceConfirmation = 0
			}
			remainingSeconds = confirmedState.RemainingSeconds - usageSinceConfirmation
			if remainingSeconds < 0 {
				remainingSeconds = 0
			}
		}
		input.Quota[now.Weekday()] = time.Duration(remainingSeconds) * time.Second
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

func establishedGraphicalSessionID(sessions []session.Session) string {
	for _, current := range sessions {
		if current.State == "active" {
			return current.BalanceAuthorizationID()
		}
	}
	return ""
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
