package web

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	protocol "github.com/ssergio100/compasso/protocol/v1"
	"github.com/ssergio100/compasso/server/storage"
)

func TestEventHubDeliversAndDropsSlowSubscribers(t *testing.T) {
	hub := newEventHub()
	events, unsubscribe := hub.subscribe("device-a")
	defer unsubscribe()
	if !hub.hasSubscribers("device-a") {
		t.Fatal("subscriber was not registered")
	}
	if hub.hasSubscribers("device-b") {
		t.Fatal("unrelated device reported subscribers")
	}
	hub.publish("device-a", streamEvent{Name: "status", Data: json.RawMessage(`{"online":true}`)})
	event := <-events
	if event.Name != "status" || !bytes.Contains(event.Data, []byte(`"online":true`)) {
		t.Fatalf("event=%+v", event)
	}
	unsubscribe()
	if hub.hasSubscribers("device-a") {
		t.Fatal("unsubscribed device still has subscribers")
	}
	// A slow subscriber must not block a publisher: the hub buffer holds 8
	// events and silently drops any overflow.
	slow, unsubscribeSlow := hub.subscribe("device-b")
	defer unsubscribeSlow()
	for i := 0; i < 9; i++ {
		hub.publish("device-b", streamEvent{Name: "status", Data: json.RawMessage(`{}`)})
	}
	for i := 0; i < 8; i++ {
		select {
		case <-slow:
		default:
			t.Fatal("buffered event was dropped")
		}
	}
	select {
	case <-slow:
		t.Fatal("overflow event should have been dropped")
	default:
	}
}

func TestDeviceStreamRequiresAdministrativeSession(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/admin/devices/device/stream", nil)
	response := httptest.NewRecorder()
	fixture.app.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized stream status=%d", response.Code)
	}
}

func TestDeviceStreamSendsHelloAndHeartbeatUpdates(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	fixture.login(t)
	device, err := fixture.store.CreateDevice(context.Background(), "Zorin", fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	token, err := fixture.store.IssueDeviceToken(context.Background(), device.ID, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(fixture.app)
	defer server.Close()

	request, err := http.NewRequest(http.MethodGet, server.URL+"/api/v1/admin/devices/"+device.ID+"/stream", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(fixture.sessionCookie)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("content type=%q", response.Header.Get("Content-Type"))
	}
	reader := bufio.NewReader(response.Body)
	hello := readSSEEvent(t, reader)
	if hello.name != "hello" {
		t.Fatalf("first event=%q", hello.name)
	}
	var initial deviceLiveStatus
	if err := json.Unmarshal(hello.data, &initial); err != nil {
		t.Fatalf("hello payload=%s err=%v", hello.data, err)
	}

	var quotas [7]int64
	quotas[time.Monday] = 3600
	if err := fixture.store.SaveQuotas(context.Background(), device.ID, quotas, 10, fixture.now); err != nil {
		t.Fatal(err)
	}
	heartbeatBody, _ := json.Marshal(protocol.HeartbeatRequest{
		PolicyRevision: 1, LocalDate: fixture.now.Format("2006-01-02"), SecondsUsed: 60,
	})
	heartbeatRequest, err := http.NewRequest(http.MethodPost, server.URL+protocol.HeartbeatPath, bytes.NewReader(heartbeatBody))
	if err != nil {
		t.Fatal(err)
	}
	heartbeatRequest.Header.Set("Authorization", "Bearer "+token)
	heartbeatRequest.Header.Set("X-Tempo-Device-ID", device.ID)
	heartbeatRequest.Header.Set(protocol.VersionHeader, protocol.CurrentProtocolVersion)
	heartbeatResponse, err := http.DefaultClient.Do(heartbeatRequest)
	if err != nil {
		t.Fatal(err)
	}
	heartbeatResponse.Body.Close()
	if heartbeatResponse.StatusCode != http.StatusOK {
		t.Fatalf("heartbeat status=%d", heartbeatResponse.StatusCode)
	}
	update := readSSEEvent(t, reader)
	if update.name != "status" {
		t.Fatalf("heartbeat event=%q", update.name)
	}
	var updated deviceLiveStatus
	if err := json.Unmarshal(update.data, &updated); err != nil {
		t.Fatalf("status payload=%s err=%v", update.data, err)
	}
	if !updated.Online || updated.UsedSeconds != 60 {
		t.Fatalf("heartbeat status=%+v", updated)
	}

	communicationEvent := readSSEEvent(t, reader)
	if communicationEvent.name != "communication" {
		t.Fatalf("after heartbeat expected communication event, got %q", communicationEvent.name)
	}
	var communicationLog storage.CommunicationLog
	if err := json.Unmarshal(communicationEvent.data, &communicationLog); err != nil {
		t.Fatalf("communication payload=%s err=%v", communicationEvent.data, err)
	}
	if communicationLog.DeviceID != device.ID || communicationLog.Operation != "heartbeat" {
		t.Fatalf("communication log=%+v", communicationLog)
	}
}

type sseMessage struct {
	name string
	data []byte
}

func readSSEEvent(t *testing.T, reader *bufio.Reader) sseMessage {
	t.Helper()
	var message sseMessage
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read stream: %v", err)
		}
		line = strings.TrimRight(line, "\n")
		if line == "" {
			return message
		}
		switch {
		case strings.HasPrefix(line, "event: "):
			message.name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			message.data = []byte(strings.TrimPrefix(line, "data: "))
		}
	}
}
