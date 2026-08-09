package daemon

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/policy"
	"github.com/sergio/compasso/agent/session"
	"github.com/sergio/compasso/agent/storage"
)

type fakeSessions struct {
	sessions     []session.Session
	terminated   []string
	terminateErr error
}

type fakeSynchronizationSource struct {
	graphicalSessionActive bool
	graphicalSessionID     string
}

func (f *fakeSynchronizationSource) SetGraphicalSession(active bool, sessionID string) {
	f.graphicalSessionActive = active
	f.graphicalSessionID = sessionID
}

func (f *fakeSessions) Sessions(context.Context, string) ([]session.Session, error) {
	return append([]session.Session(nil), f.sessions...), nil
}

func (f *fakeSessions) Logout(_ context.Context, current session.Session) error {
	if f.terminateErr != nil {
		return f.terminateErr
	}
	f.terminated = append(f.terminated, current.ID)
	return nil
}

func TestQuotaExpiryTerminatesGraphicalSession(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 3*time.Second)); err != nil {
		t.Fatal(err)
	}
	sessions := &fakeSessions{sessions: []session.Session{{
		ID: "3", User: "child", Type: "wayland", Class: "user", State: "active",
	}}}
	daemon, err := New(store, sessions, "child", time.Second)
	if err != nil {
		t.Fatal(err)
	}

	status, err := daemon.Step(ctx, start)
	if err != nil || !status.Decision.Allowed {
		t.Fatalf("initial step = %+v, err=%v", status, err)
	}
	status, err = daemon.Step(ctx, start.Add(2*time.Second))
	if err != nil || status.UsageSeconds != 2 || !status.Decision.Allowed {
		t.Fatalf("second step = %+v, err=%v", status, err)
	}
	status, err = daemon.Step(ctx, start.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if status.Decision.Allowed || status.Decision.Reason != policy.ReasonQuota || status.UsageSeconds != 3 {
		t.Fatalf("expiry status = %+v", status)
	}
	if len(sessions.terminated) != 1 || sessions.terminated[0] != "3" {
		t.Fatalf("terminated sessions = %v, want [3]", sessions.terminated)
	}
	durable, err := store.LoadDailyUsage(ctx, start.Format("2006-01-02"))
	if err != nil || durable.SecondsUsed != 3 {
		t.Fatalf("durable usage at block = %+v, err=%v", durable, err)
	}

	// The same still-closing session is not sent to logind repeatedly.
	if _, err := daemon.Step(ctx, start.Add(4*time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(sessions.terminated) != 1 {
		t.Fatalf("same blocked session terminated repeatedly: %v", sessions.terminated)
	}
}

func TestDelayedCycleDoesNotCountPastQuotaExpiry(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 3*time.Second)); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := daemon.Step(ctx, start.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if status.UsageSeconds != 3 || status.Decision.Reason != policy.ReasonQuota || len(sessions.terminated) != 1 {
		t.Fatalf("delayed expiry status=%+v terminated=%v", status, sessions.terminated)
	}
}

func TestSynchronizedDaemonConsumesConfirmedBalanceWithoutRecalculatingQuota(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	// The copied policy deliberately says eight hours. In synchronized mode the
	// three-second server anchor is the only balance authority.
	if err := store.ReplacePolicy(ctx, testPolicy(4, start.Weekday(), 8*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConfirmedSessionState(ctx, storage.ConfirmedSessionState{
		Revision: 4, SessionID: "3", LocalDate: start.Format("2006-01-02"),
		RemainingSeconds: 3, UsageSeconds: 0, ConfirmedAt: start,
	}); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	synchronization := &fakeSynchronizationSource{}
	policyDaemon, err := New(store, sessions, "child", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	policyDaemon.SetSynchronizationSource(synchronization)

	first, err := policyDaemon.Step(ctx, start)
	if err != nil || !first.Decision.Allowed || first.Decision.Remaining != 3*time.Second {
		t.Fatalf("first confirmed decision=%+v err=%v", first, err)
	}
	second, err := policyDaemon.Step(ctx, start.Add(2*time.Second))
	if err != nil || second.Decision.Remaining != time.Second || second.UsageSeconds != 2 {
		t.Fatalf("decremented confirmed decision=%+v err=%v", second, err)
	}
	expired, err := policyDaemon.Step(ctx, start.Add(3*time.Second))
	if err != nil || expired.Decision.Reason != policy.ReasonQuota || len(sessions.terminated) != 1 {
		t.Fatalf("confirmed balance expiry=%+v terminated=%v err=%v", expired, sessions.terminated, err)
	}
	if !synchronization.graphicalSessionActive || synchronization.graphicalSessionID != "3" {
		t.Fatalf("reported graphical session active=%t id=%q", synchronization.graphicalSessionActive, synchronization.graphicalSessionID)
	}
}

func TestQuotaCycleEmitsOneMinuteDesktopAlert(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	snapshot := testPolicy(1, start.Weekday(), 2*time.Minute)
	snapshot.WarningMinutes = 1
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	policyDaemon, err := New(store, graphicalFake(), "child", time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := policyDaemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := policyDaemon.Step(ctx, start.Add(time.Minute))
	if err != nil || len(status.DueAlerts) != 1 || status.DueAlerts[0].Kind != "one_minute" {
		t.Fatalf("status due alerts=%+v err=%v", status.DueAlerts, err)
	}
}

func TestRoutineStartTerminatesSession(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 21, 59, 59, 0, time.Local)
	snapshot := testPolicy(1, start.Weekday(), time.Hour)
	snapshot.Routines = []storage.StoredRoutine{{
		ID: "sleep", Name: "Sleep", Days: daySet(start.Weekday()),
		Start: 22 * time.Hour, End: 8 * time.Hour, Enabled: true,
	}}
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := daemon.Step(ctx, start.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if status.Decision.Reason != policy.ReasonRoutine || len(sessions.terminated) != 1 {
		t.Fatalf("routine status=%+v terminated=%v", status, sessions.terminated)
	}
}

func TestPausedMonitoringDoesNotCountOrTerminate(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	snapshot := testPolicy(1, start.Weekday(), 0)
	snapshot.MonitoringPaused = true
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := daemon.Step(ctx, start.Add(10*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if !status.Decision.Allowed || status.Decision.Reason != policy.ReasonPaused || status.UsageSeconds != 0 {
		t.Fatalf("paused status = %+v", status)
	}
	if len(sessions.terminated) != 0 {
		t.Fatalf("paused monitoring terminated sessions: %v", sessions.terminated)
	}
}

func TestPauseBeforeExpiryStopsCounterAndPreventsLogout(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 3*time.Second)); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := daemon.Step(ctx, start.Add(2*time.Second))
	if err != nil || status.UsageSeconds != 2 {
		t.Fatalf("usage before pause = %+v, err=%v", status, err)
	}
	paused := testPolicy(2, start.Weekday(), 3*time.Second)
	paused.MonitoringPaused = true
	if err := store.ReplacePolicy(ctx, paused); err != nil {
		t.Fatal(err)
	}
	status, err = daemon.Step(ctx, start.Add(2500*time.Millisecond))
	if err != nil || !status.Decision.Allowed || status.Decision.Reason != policy.ReasonPaused {
		t.Fatalf("pause transition = %+v, err=%v", status, err)
	}
	status, err = daemon.Step(ctx, start.Add(10*time.Second))
	if err != nil || status.UsageSeconds != 2 || len(sessions.terminated) != 0 {
		t.Fatalf("after pause = %+v, err=%v terminated=%v", status, err, sessions.terminated)
	}
}

func TestNoGraphicalSessionDoesNotCount(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), time.Hour)); err != nil {
		t.Fatal(err)
	}
	sessions := &fakeSessions{sessions: []session.Session{{
		ID: "4", User: "child", Type: "tty", Class: "user", State: "active",
	}}}
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := daemon.Step(ctx, start.Add(30*time.Second))
	if err != nil || status.UsageSeconds != 0 {
		t.Fatalf("non-graphical status=%+v err=%v", status, err)
	}
}

func TestBlockedReloginWaitsForActiveSessionToStabilize(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 0)); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	if _, err := daemon.Step(ctx, start.Add(blockedReloginStabilization-time.Second)); err != nil {
		t.Fatal(err)
	}
	if len(sessions.terminated) != 0 {
		t.Fatalf("new session terminated during desktop startup: %v", sessions.terminated)
	}
	if _, err := daemon.Step(ctx, start.Add(blockedReloginStabilization)); err != nil {
		t.Fatal(err)
	}
	if len(sessions.terminated) != 1 || sessions.terminated[0] != "3" {
		t.Fatalf("stabilized blocked session was not terminated: %v", sessions.terminated)
	}
}

func TestBlockedReloginWaitsUntilLogindReportsActive(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 0)); err != nil {
		t.Fatal(err)
	}
	sessions := &fakeSessions{sessions: []session.Session{{
		ID: "5", User: "child", Type: "wayland", Class: "user", State: "opening",
	}}}
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	if _, err := daemon.Step(ctx, start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(sessions.terminated) != 0 {
		t.Fatalf("opening session terminated: %v", sessions.terminated)
	}
	sessions.sessions[0].State = "active"
	if _, err := daemon.Step(ctx, start.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := daemon.Step(ctx, start.Add(time.Minute+blockedReloginStabilization)); err != nil {
		t.Fatal(err)
	}
	if len(sessions.terminated) != 1 || sessions.terminated[0] != "5" {
		t.Fatalf("active stabilized session was not terminated: %v", sessions.terminated)
	}
}

func TestBlockedLoginWaitsForConfirmedSessionState(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 0)); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	synchronization := &fakeSynchronizationSource{}
	policyDaemon, _ := New(store, sessions, "child", time.Second)
	policyDaemon.SetSynchronizationSource(synchronization)

	status, err := policyDaemon.Step(ctx, start)
	if err != nil || !status.AwaitingSynchronization || len(sessions.terminated) != 0 {
		t.Fatalf("initial blocked login status=%+v err=%v terminated=%v", status, err, sessions.terminated)
	}
	status, err = policyDaemon.Step(ctx, start.Add(blockedReloginStabilization))
	if err != nil || !status.AwaitingSynchronization || len(sessions.terminated) != 0 {
		t.Fatalf("offline blocked login status=%+v err=%v terminated=%v", status, err, sessions.terminated)
	}

	if err := store.SaveConfirmedSessionState(ctx, storage.ConfirmedSessionState{
		Revision: 1, SessionID: "3", LocalDate: start.Format("2006-01-02"),
		RemainingSeconds: 0, UsageSeconds: 0, ConfirmedAt: start.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	status, err = policyDaemon.Step(ctx, start.Add(blockedReloginStabilization+time.Second))
	if err != nil || status.AwaitingSynchronization || len(sessions.terminated) != 1 {
		t.Fatalf("confirmed blocked login status=%+v err=%v terminated=%v", status, err, sessions.terminated)
	}
}

func TestBlockedLoginRemainsOpenWhenConfirmedStateAddsTime(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 0)); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	synchronization := &fakeSynchronizationSource{}
	policyDaemon, _ := New(store, sessions, "child", time.Second)
	policyDaemon.SetSynchronizationSource(synchronization)
	if _, err := policyDaemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}

	if err := store.ReplacePolicy(ctx, testPolicy(2, start.Weekday(), time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveConfirmedSessionState(ctx, storage.ConfirmedSessionState{
		Revision: 2, SessionID: "3", LocalDate: start.Format("2006-01-02"),
		RemainingSeconds: 60, UsageSeconds: 0, ConfirmedAt: start.Add(time.Second),
	}); err != nil {
		t.Fatal(err)
	}
	status, err := policyDaemon.Step(ctx, start.Add(time.Second))
	if err != nil || !status.Decision.Allowed || status.AwaitingSynchronization || len(sessions.terminated) != 0 {
		t.Fatalf("updated blocked login status=%+v err=%v terminated=%v", status, err, sessions.terminated)
	}
}

func TestMidnightSplitsUsageBetweenLocalDates(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	location := time.FixedZone("test", -3*60*60)
	start := time.Date(2026, time.August, 9, 23, 59, 59, 0, location)
	snapshot := testPolicy(1, start.Weekday(), time.Hour)
	snapshot.WeeklyQuota[start.AddDate(0, 0, 1).Weekday()] = time.Hour
	if err := store.ReplacePolicy(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := daemon.Step(ctx, start.Add(2*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	oldDay, err := store.LoadDailyUsage(ctx, "2026-08-09")
	if err != nil {
		t.Fatal(err)
	}
	if oldDay.SecondsUsed != 1 || status.UsageSeconds != 1 {
		t.Fatalf("usage split: old=%d new=%d, want 1 and 1", oldDay.SecondsUsed, status.UsageSeconds)
	}
}

func TestLogoutFailureDoesNotDoubleCountElapsedTime(t *testing.T) {
	ctx := context.Background()
	store := testStore(t)
	defer store.Close()
	start := time.Date(2026, time.August, 10, 14, 0, 0, 0, time.Local)
	if err := store.ReplacePolicy(ctx, testPolicy(1, start.Weekday(), 3*time.Second)); err != nil {
		t.Fatal(err)
	}
	sessions := graphicalFake()
	sessions.terminateErr = errors.New("temporary logind failure")
	daemon, _ := New(store, sessions, "child", time.Second)
	if _, err := daemon.Step(ctx, start); err != nil {
		t.Fatal(err)
	}
	status, err := daemon.Step(ctx, start.Add(3*time.Second))
	if err == nil || status.UsageSeconds != 3 {
		t.Fatalf("failed logout status=%+v err=%v", status, err)
	}
	sessions.terminateErr = nil
	status, err = daemon.Step(ctx, start.Add(4*time.Second))
	if err != nil || status.UsageSeconds != 3 || len(sessions.terminated) != 1 {
		t.Fatalf("retry status=%+v err=%v terminated=%v", status, err, sessions.terminated)
	}
}

func testStore(t *testing.T) *storage.Store {
	t.Helper()
	store, err := storage.Open(context.Background(), filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func testPolicy(revision int64, weekday time.Weekday, quota time.Duration) storage.PolicySnapshot {
	var quotas [7]time.Duration
	quotas[weekday] = quota
	return storage.PolicySnapshot{
		Revision: revision, WeeklyQuota: quotas,
		UpdatedAt: time.Date(2026, time.August, 10, 10, 0, 0, 0, time.UTC),
	}
}

func graphicalFake() *fakeSessions {
	return &fakeSessions{sessions: []session.Session{{
		ID: "3", User: "child", Type: "wayland", Class: "user", State: "active",
	}}}
}

func daySet(days ...time.Weekday) (result [7]bool) {
	for _, day := range days {
		result[day] = true
	}
	return result
}
