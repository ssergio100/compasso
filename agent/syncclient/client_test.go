package syncclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/localauth"
	"github.com/sergio/compasso/agent/policy"
	agentstorage "github.com/sergio/compasso/agent/storage"
	serverstorage "github.com/sergio/compasso/server/storage"
	"github.com/sergio/compasso/server/web"
)

func TestRevisionOfflineQueueAndImmediateEnforcement(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.Local)
	serverStore, err := serverstorage.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer serverStore.Close()
	device, err := serverStore.CreateDevice(ctx, "Zorin", now)
	if err != nil {
		t.Fatal(err)
	}
	token, err := serverStore.IssueDeviceToken(ctx, device.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	var quotas [7]int64
	quotas[time.Monday] = 3600
	// Initial revision is 1; nine writes produce the explicit revision 10 case.
	for revision := 2; revision <= 10; revision++ {
		if err := serverStore.SaveQuotas(ctx, device.ID, quotas, 10, now.Add(time.Duration(revision)*time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	application, err := web.NewWithAssets(serverStore, false, time.Hour, "../../server/web")
	if err != nil {
		t.Fatal(err)
	}
	var online uint32
	atomic.StoreUint32(&online, 1)
	proxyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadUint32(&online) == 0 {
			http.Error(w, "offline", http.StatusServiceUnavailable)
			return
		}
		application.ServeHTTP(w, r)
	})
	httpClient := &http.Client{Transport: handlerTransport{handler: proxyHandler}}
	agentStore, err := agentstorage.Open(ctx, filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer agentStore.Close()
	client, err := New(agentStore, httpClient, Config{
		ServerURL: "http://tempo.test", DeviceID: device.ID, DeviceToken: token, HeartbeatInterval: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Heartbeat(ctx, now); err != nil {
		t.Fatal(err)
	}
	localPolicy, err := agentStore.LoadPolicy(ctx)
	if err != nil || localPolicy.Revision != 10 {
		t.Fatalf("initial revision=%d err=%v", localPolicy.Revision, err)
	}
	if err := agentStore.CheckpointUsage(ctx, agentstorage.DailyUsage{LocalDate: "2026-08-10", SecondsUsed: 120, CheckpointAt: now}); err != nil {
		t.Fatal(err)
	}
	quotas[time.Monday] = 60
	if err := serverStore.SaveQuotas(ctx, device.ID, quotas, 10, now.Add(20*time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Heartbeat(ctx, now.Add(21*time.Second)); err != nil {
		t.Fatal(err)
	}
	localPolicy, err = agentStore.LoadPolicy(ctx)
	if err != nil || localPolicy.Revision != 11 || localPolicy.WeeklyQuota[time.Monday] != time.Minute {
		t.Fatalf("revision 11 not applied immediately: policy=%+v err=%v", localPolicy, err)
	}
	decision, err := policy.Evaluate(policy.Input{
		Now: now.Add(21 * time.Second), Monitoring: policy.MonitoringActive,
		Quota: policy.WeeklyQuota(localPolicy.WeeklyQuota), Consumed: 120 * time.Second,
	})
	if err != nil || decision.Allowed {
		t.Fatalf("reduced remote quota did not block immediately: decision=%+v err=%v", decision, err)
	}

	bonusPayload, _ := json.Marshal(map[string]interface{}{"local_date": "2026-08-10", "seconds": 300, "origin": "local"})
	bonus := agentstorage.Bonus{UUID: "offline-bonus", LocalDate: "2026-08-10", Seconds: 300, Origin: "local", CreatedAt: now.Add(time.Minute)}
	event := agentstorage.PendingEvent{UUID: bonus.UUID, Kind: "bonus_added", PayloadJSON: string(bonusPayload), CreatedAt: bonus.CreatedAt}
	if err := agentStore.AddBonusWithEvent(ctx, bonus, event); err != nil {
		t.Fatal(err)
	}
	atomic.StoreUint32(&online, 0)
	if _, err := client.Heartbeat(ctx, now.Add(2*time.Minute)); err == nil {
		t.Fatal("offline heartbeat unexpectedly succeeded")
	}
	tracker, err := agentstorage.NewUsageTracker(ctx, agentStore, "2026-08-10", 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := tracker.Add(ctx, 30*time.Minute, now.Add(32*time.Minute)); err != nil {
		t.Fatal(err)
	}
	preserved, err := agentStore.LoadPolicy(ctx)
	usage, usageErr := agentStore.LoadDailyUsage(ctx, "2026-08-10")
	if err != nil || usageErr != nil || preserved.Revision != 11 || usage.SecondsUsed != 1920 {
		t.Fatalf("offline state not preserved: revision=%d usage=%d errors=%v/%v", preserved.Revision, usage.SecondsUsed, err, usageErr)
	}
	pending, err := agentStore.PendingEvents(ctx, 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("offline event queue=%+v err=%v", pending, err)
	}
	atomic.StoreUint32(&online, 1)
	if _, err := client.Heartbeat(ctx, now.Add(33*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Heartbeat(ctx, now.Add(34*time.Minute)); err != nil {
		t.Fatal(err)
	}
	pending, err = agentStore.PendingEvents(ctx, 10)
	serverSummary, summaryErr := serverStore.LoadDailySummary(ctx, device.ID, "2026-08-10")
	if err != nil || summaryErr != nil || len(pending) != 0 || serverSummary.BonusSeconds != 300 {
		t.Fatalf("reconnected bonus pending=%d server=%+v errors=%v/%v", len(pending), serverSummary, err, summaryErr)
	}
}

func TestTransportErrorsRedactDeviceToken(t *testing.T) {
	ctx := context.Background()
	agentStore, err := agentstorage.Open(ctx, filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer agentStore.Close()
	deviceToken := "secret-device-token-that-must-not-appear"
	httpClient := &http.Client{Transport: failingTransport{failure: fmt.Errorf("failed Authorization: Bearer %s", deviceToken)}}
	client, err := New(agentStore, httpClient, Config{
		ServerURL: "http://tempo.test", DeviceID: "device", DeviceToken: deviceToken,
		HeartbeatInterval: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, heartbeatError := client.Heartbeat(ctx, time.Date(2026, time.August, 10, 12, 0, 0, 0, time.Local))
	if heartbeatError == nil || strings.Contains(heartbeatError.Error(), deviceToken) || !strings.Contains(heartbeatError.Error(), "[REDACTED]") {
		t.Fatalf("transport error was not sanitized: %v", heartbeatError)
	}
}

func TestLocalPasswordChangesOnlyAfterSuccessfulSynchronization(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.Local)
	serverStore, err := serverstorage.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer serverStore.Close()
	device, err := serverStore.CreateDevice(ctx, "Password sync", now)
	if err != nil {
		t.Fatal(err)
	}
	deviceToken, err := serverStore.IssueDeviceToken(ctx, device.ID, now)
	if err != nil {
		t.Fatal(err)
	}
	passwordParameters := localauth.Argon2Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16}
	oldVerifier, err := localauth.HashPassword("old-password", passwordParameters)
	if err != nil {
		t.Fatal(err)
	}
	newVerifier, err := localauth.HashPassword("new-password", passwordParameters)
	if err != nil {
		t.Fatal(err)
	}
	if err := serverStore.SetLocalPassword(ctx, device.ID, oldVerifier, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	application, err := web.NewWithAssets(serverStore, false, time.Hour, "../../server/web")
	if err != nil {
		t.Fatal(err)
	}
	var serverOnline uint32 = 1
	httpClient := &http.Client{Transport: handlerTransport{handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadUint32(&serverOnline) == 0 {
			http.Error(w, "offline", http.StatusServiceUnavailable)
			return
		}
		application.ServeHTTP(w, r)
	})}}
	agentStore, err := agentstorage.Open(ctx, filepath.Join(t.TempDir(), "agent.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer agentStore.Close()
	synchronizationClient, err := New(agentStore, httpClient, Config{
		ServerURL: "http://tempo.test", DeviceID: device.ID,
		DeviceToken: deviceToken, HeartbeatInterval: 10 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := synchronizationClient.Heartbeat(ctx, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}

	if err := serverStore.SetLocalPassword(ctx, device.ID, newVerifier, now.Add(3*time.Second)); err != nil {
		t.Fatal(err)
	}
	atomic.StoreUint32(&serverOnline, 0)
	if _, err := synchronizationClient.Heartbeat(ctx, now.Add(4*time.Second)); err == nil {
		t.Fatal("offline password synchronization unexpectedly succeeded")
	}
	offlinePolicy, err := agentStore.LoadPolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	oldPasswordValid, _ := localauth.VerifyPassword("old-password", offlinePolicy.LocalPasswordVerifier)
	newPasswordValidOffline, _ := localauth.VerifyPassword("new-password", offlinePolicy.LocalPasswordVerifier)
	if !oldPasswordValid || newPasswordValidOffline {
		t.Fatal("offline client did not preserve the old password verifier")
	}

	atomic.StoreUint32(&serverOnline, 1)
	if _, err := synchronizationClient.Heartbeat(ctx, now.Add(5*time.Second)); err != nil {
		t.Fatal(err)
	}
	onlinePolicy, err := agentStore.LoadPolicy(ctx)
	if err != nil {
		t.Fatal(err)
	}
	oldPasswordValid, _ = localauth.VerifyPassword("old-password", onlinePolicy.LocalPasswordVerifier)
	newPasswordValid, _ := localauth.VerifyPassword("new-password", onlinePolicy.LocalPasswordVerifier)
	if oldPasswordValid || !newPasswordValid {
		t.Fatal("successful synchronization did not replace the password verifier")
	}
}

type handlerTransport struct{ handler http.Handler }

func (transport handlerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	recorder := httptest.NewRecorder()
	transport.handler.ServeHTTP(recorder, request)
	return recorder.Result(), nil
}

type failingTransport struct{ failure error }

func (transport failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if transport.failure == nil {
		return nil, errors.New("transport failed")
	}
	return nil, transport.failure
}
