package web

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func FuzzHeartbeatPayloadDoesNotPanic(fuzzer *testing.F) {
	fixture := newWebFixture(fuzzer, false, time.Hour)
	defer fixture.store.Close()
	device, err := fixture.store.CreateDevice(context.Background(), "Fuzz device", fixture.now)
	if err != nil {
		fuzzer.Fatal(err)
	}
	deviceToken, err := fixture.store.IssueDeviceToken(context.Background(), device.ID, fixture.now)
	if err != nil {
		fuzzer.Fatal(err)
	}
	fuzzer.Add([]byte(`{}`))
	fuzzer.Add([]byte(`{"policy_revision":-1,"local_date":"invalid","seconds_used":-1}`))
	fuzzer.Add([]byte(`{"unknown":true}`))
	fuzzer.Add([]byte(`{"events":[{"uuid":"x","kind":"bonus_added","payload":null}]}`))
	fuzzer.Fuzz(func(t *testing.T, payload []byte) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/device/heartbeat", bytes.NewReader(payload))
		request.Header.Set(deviceIDHeader, device.ID)
		request.Header.Set("Authorization", "Bearer "+deviceToken)
		response := httptest.NewRecorder()
		fixture.app.ServeHTTP(response, request)
		if response.Code < 200 || response.Code > 599 {
			t.Fatalf("invalid HTTP status %d", response.Code)
		}
	})
}
