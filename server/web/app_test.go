package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/localauth"
	protocol "github.com/sergio/compasso/protocol/v1"
	serverstorage "github.com/sergio/compasso/server/storage"
)

type webFixture struct {
	app           *App
	store         *serverstorage.Store
	now           time.Time
	sessionCookie *http.Cookie
	csrfToken     string
}

func TestAdministrativeSessionCORSAndSecureCookie(t *testing.T) {
	fixture := newWebFixture(t, true, time.Hour)
	defer fixture.store.Close()
	adminOrigin := "https://admin.example"
	if err := fixture.app.SetAdminOrigin(adminOrigin); err != nil {
		t.Fatal(err)
	}

	forbiddenRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	forbiddenRequest.Header.Set("Origin", "https://attacker.example")
	forbiddenResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("forbidden origin status=%d", forbiddenResponse.Code)
	}

	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	sessionRequest.Header.Set("Origin", adminOrigin)
	sessionResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(sessionResponse, sessionRequest)
	if sessionResponse.Code != http.StatusOK || sessionResponse.Header().Get("Access-Control-Allow-Origin") != adminOrigin {
		t.Fatalf("session bootstrap status=%d headers=%v", sessionResponse.Code, sessionResponse.Header())
	}
	loginCSRFTokenCookie := findCookie(t, sessionResponse.Result().Cookies(), loginCSRFCookie)
	var anonymousSession adminSessionResponse
	decodeResponse(t, sessionResponse, &anonymousSession)

	wrongLoginBody, _ := json.Marshal(adminLoginRequest{
		Login: "admin", Password: "wrong", CSRFToken: anonymousSession.CSRFToken,
	})
	wrongLoginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/session", bytes.NewReader(wrongLoginBody))
	wrongLoginRequest.Header.Set("Content-Type", "application/json")
	wrongLoginRequest.AddCookie(loginCSRFTokenCookie)
	wrongLoginResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(wrongLoginResponse, wrongLoginRequest)
	if wrongLoginResponse.Code != http.StatusUnauthorized || strings.Contains(wrongLoginResponse.Body.String(), "admin") {
		t.Fatalf("wrong login status=%d body=%s", wrongLoginResponse.Code, wrongLoginResponse.Body.String())
	}

	loginBody, _ := json.Marshal(adminLoginRequest{
		Login: "admin", Password: "secret", CSRFToken: anonymousSession.CSRFToken,
	})
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/session", bytes.NewReader(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.Header.Set("Origin", adminOrigin)
	loginRequest.AddCookie(loginCSRFTokenCookie)
	loginResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("API login status=%d body=%s", loginResponse.Code, loginResponse.Body.String())
	}
	adminSessionCookie := findCookie(t, loginResponse.Result().Cookies(), sessionCookieName)
	if !adminSessionCookie.HttpOnly || !adminSessionCookie.Secure || adminSessionCookie.SameSite != http.SameSiteStrictMode ||
		adminSessionCookie.Path != "/api/v1/admin" {
		t.Fatalf("unsafe session cookie: %+v", adminSessionCookie)
	}
	if loginResponse.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("production API response is missing HSTS")
	}
}

func TestSameHostAdministrativeOriginForLocalNetworkInstallation(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	if err := fixture.app.SetAdminOrigin("same-host"); err != nil {
		t.Fatal(err)
	}

	allowedRequest := httptest.NewRequest(http.MethodGet, "http://192.168.1.20:8181/api/v1/admin/session", nil)
	allowedRequest.Header.Set("Origin", "http://192.168.1.20:8182")
	allowedResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(allowedResponse, allowedRequest)
	if allowedResponse.Code != http.StatusOK || allowedResponse.Header().Get("Access-Control-Allow-Origin") != "http://192.168.1.20:8182" {
		t.Fatalf("same-host origin rejected: status=%d headers=%v", allowedResponse.Code, allowedResponse.Header())
	}

	forbiddenRequest := httptest.NewRequest(http.MethodGet, "http://192.168.1.20:8181/api/v1/admin/session", nil)
	forbiddenRequest.Header.Set("Origin", "http://192.168.1.30:8182")
	forbiddenResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(forbiddenResponse, forbiddenRequest)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("different-host origin status=%d", forbiddenResponse.Code)
	}
}

func TestInitialAdministratorIsConfiguredAfterInstallation(t *testing.T) {
	ctx := context.Background()
	store, err := serverstorage.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	application, err := New(store, false, time.Hour, time.Minute, "")
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time { return time.Date(2026, time.August, 10, 12, 0, 0, 0, time.UTC) }

	sessionRequest := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	sessionResponse := httptest.NewRecorder()
	application.ServeHTTP(sessionResponse, sessionRequest)
	var setupSession adminSessionResponse
	decodeResponse(t, sessionResponse, &setupSession)
	if !setupSession.SetupRequired || setupSession.Authenticated {
		t.Fatalf("fresh installation session=%+v", setupSession)
	}
	setupCSRFCookie := findCookie(t, sessionResponse.Result().Cookies(), loginCSRFCookie)

	setupBody, _ := json.Marshal(adminSetupRequest{
		Login: "sergio", Password: "senha-memoravel", PasswordConfirmation: "senha-memoravel",
		CSRFToken: setupSession.CSRFToken,
	})
	setupRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/setup", bytes.NewReader(setupBody))
	setupRequest.Header.Set("Content-Type", "application/json")
	setupRequest.AddCookie(setupCSRFCookie)
	setupResponse := httptest.NewRecorder()
	application.ServeHTTP(setupResponse, setupRequest)
	if setupResponse.Code != http.StatusCreated {
		t.Fatalf("initial setup status=%d body=%s", setupResponse.Code, setupResponse.Body.String())
	}
	var authenticatedSession adminSessionResponse
	decodeResponse(t, setupResponse, &authenticatedSession)
	if !authenticatedSession.Authenticated || authenticatedSession.Login != "sergio" || authenticatedSession.SetupRequired {
		t.Fatalf("created session=%+v", authenticatedSession)
	}
	findCookie(t, setupResponse.Result().Cookies(), sessionCookieName)

	administrator, err := store.AdminByLogin(ctx, "sergio")
	if err != nil {
		t.Fatal(err)
	}
	passwordMatches, err := localauth.VerifyPassword("senha-memoravel", administrator.PasswordHash)
	if err != nil || !passwordMatches {
		t.Fatalf("configured password was not stored securely: matches=%t err=%v", passwordMatches, err)
	}

	secondSetupRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/setup", bytes.NewReader(setupBody))
	secondSetupRequest.Header.Set("Content-Type", "application/json")
	secondSetupRequest.AddCookie(setupCSRFCookie)
	secondSetupResponse := httptest.NewRecorder()
	application.ServeHTTP(secondSetupResponse, secondSetupRequest)
	if secondSetupResponse.Code != http.StatusConflict {
		t.Fatalf("second setup status=%d body=%s", secondSetupResponse.Code, secondSetupResponse.Body.String())
	}
}

func TestAdministrativeJSONAPIWorkflow(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	fixture.login(t)

	missingCSRFResponse := fixture.requestJSON(
		http.MethodPost, "/api/v1/admin/devices", createAdminDeviceRequest{Name: "PC"}, false,
	)
	if missingCSRFResponse.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status=%d", missingCSRFResponse.Code)
	}

	createdResponse := fixture.requestJSON(
		http.MethodPost, "/api/v1/admin/devices", createAdminDeviceRequest{Name: "Debian KDE"}, true,
	)
	if createdResponse.Code != http.StatusCreated {
		t.Fatalf("create device status=%d body=%s", createdResponse.Code, createdResponse.Body.String())
	}
	var createdDevice adminDeviceResponse
	decodeResponse(t, createdResponse, &createdDevice)
	devicePath := "/api/v1/admin/devices/" + createdDevice.ID

	policyResponse := fixture.requestJSON(http.MethodPut, devicePath+"/policy", updateAdminPolicyRequest{
		WeeklyQuota: [7]int64{0, 3600, 7200, 0, 0, 0, 0}, WarningMinutes: 5,
	}, true)
	if policyResponse.Code != http.StatusOK {
		t.Fatalf("update policy status=%d body=%s", policyResponse.Code, policyResponse.Body.String())
	}

	routineResponse := fixture.requestJSON(http.MethodPost, devicePath+"/routines", saveAdminRoutineRequest{
		Name: "Dormir", Days: [7]bool{false, true, true, true, true, true, false},
		Start: 22 * 3600, End: 8 * 3600, Enabled: true,
	}, true)
	if routineResponse.Code != http.StatusCreated {
		t.Fatalf("create routine status=%d body=%s", routineResponse.Code, routineResponse.Body.String())
	}
	var createdRoutine map[string]string
	decodeResponse(t, routineResponse, &createdRoutine)
	routineID := createdRoutine["id"]
	updatedRoutineResponse := fixture.requestJSON(http.MethodPut, devicePath+"/routines/"+routineID, saveAdminRoutineRequest{
		Name: "Dormir cedo", Days: [7]bool{false, true, true, true, true, true, false},
		Start: 21 * 3600, End: 8 * 3600, Enabled: true,
	}, true)
	if updatedRoutineResponse.Code != http.StatusOK {
		t.Fatalf("update routine status=%d body=%s", updatedRoutineResponse.Code, updatedRoutineResponse.Body.String())
	}

	password := "new responsible secret"
	passwordResponse := fixture.requestJSON(http.MethodPut, devicePath+"/password", updateAdminPasswordRequest{
		Password: password, PasswordConfirmation: password,
	}, true)
	if passwordResponse.Code != http.StatusOK || strings.Contains(passwordResponse.Body.String(), password) ||
		strings.Contains(passwordResponse.Body.String(), "$argon2id$") {
		t.Fatalf("password response status=%d body=%s", passwordResponse.Code, passwordResponse.Body.String())
	}

	tokenResponse := fixture.requestJSON(http.MethodPost, devicePath+"/token", nil, true)
	if tokenResponse.Code != http.StatusCreated {
		t.Fatalf("issue token status=%d body=%s", tokenResponse.Code, tokenResponse.Body.String())
	}
	var issuedToken map[string]string
	decodeResponse(t, tokenResponse, &issuedToken)
	if issuedToken["device_token"] == "" {
		t.Fatal("issued device token is empty")
	}

	bonusResponse := fixture.requestJSON(
		http.MethodPost, devicePath+"/bonus", addAdminBonusRequest{Minutes: 10}, true,
	)
	if bonusResponse.Code != http.StatusAccepted {
		t.Fatalf("queue bonus status=%d body=%s", bonusResponse.Code, bonusResponse.Body.String())
	}
	commandResponse := fixture.requestJSON(http.MethodPost, devicePath+"/commands", queueAdminCommandRequest{
		Command: "pause_monitoring",
	}, true)
	if commandResponse.Code != http.StatusAccepted {
		t.Fatalf("queue command status=%d body=%s", commandResponse.Code, commandResponse.Body.String())
	}

	detailResponse := fixture.requestJSON(http.MethodGet, devicePath, nil, false)
	if detailResponse.Code != http.StatusOK || strings.Contains(detailResponse.Body.String(), "local_password_verifier") ||
		strings.Contains(detailResponse.Body.String(), "$argon2id$") || strings.Contains(detailResponse.Body.String(), issuedToken["device_token"]) {
		t.Fatalf("device detail leaked secret or failed: status=%d body=%s", detailResponse.Code, detailResponse.Body.String())
	}
	var detail adminDeviceDetailResponse
	decodeResponse(t, detailResponse, &detail)
	if detail.Policy.WeeklyQuota[time.Monday] != 3600 || detail.Policy.WarningMinutes != 5 ||
		len(detail.Policy.Routines) != 1 || !detail.Policy.PasswordSet || !detail.Policy.MonitoringPaused {
		t.Fatalf("unexpected device detail: %+v", detail)
	}

	statusResponse := fixture.requestJSON(http.MethodGet, devicePath+"/status", nil, false)
	var liveStatus deviceLiveStatus
	decodeResponse(t, statusResponse, &liveStatus)
	if statusResponse.Code != http.StatusOK || liveStatus.LocalDate != "2026-08-10" ||
		liveStatus.TodayQuotaSeconds != 3600 || liveStatus.RemainingSeconds != 4200 {
		t.Fatalf("unexpected live status: code=%d value=%+v", statusResponse.Code, liveStatus)
	}
	eventsResponse := fixture.requestJSON(http.MethodGet, devicePath+"/events?limit=20", nil, false)
	if eventsResponse.Code != http.StatusOK || !strings.Contains(eventsResponse.Body.String(), "local_password_changed") {
		t.Fatalf("events status=%d body=%s", eventsResponse.Code, eventsResponse.Body.String())
	}

	renameResponse := fixture.requestJSON(
		http.MethodPatch, devicePath, updateAdminDeviceRequest{Name: "PC do quarto"}, true,
	)
	if renameResponse.Code != http.StatusOK {
		t.Fatalf("rename status=%d body=%s", renameResponse.Code, renameResponse.Body.String())
	}
	deleteRoutineResponse := fixture.requestJSON(http.MethodDelete, devicePath+"/routines/"+routineID, nil, true)
	if deleteRoutineResponse.Code != http.StatusNoContent {
		t.Fatalf("delete routine status=%d", deleteRoutineResponse.Code)
	}
	revokeTokenResponse := fixture.requestJSON(http.MethodDelete, devicePath+"/token", nil, true)
	if revokeTokenResponse.Code != http.StatusNoContent {
		t.Fatalf("revoke token status=%d", revokeTokenResponse.Code)
	}
	listResponse := fixture.requestJSON(http.MethodGet, "/api/v1/admin/devices", nil, false)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "PC do quarto") {
		t.Fatalf("list devices status=%d body=%s", listResponse.Code, listResponse.Body.String())
	}
	deleteDeviceResponse := fixture.requestJSON(http.MethodDelete, devicePath, nil, true)
	if deleteDeviceResponse.Code != http.StatusNoContent {
		t.Fatalf("delete device status=%d", deleteDeviceResponse.Code)
	}
}

func TestExpiredAdministrativeSessionReturnsUnauthorized(t *testing.T) {
	fixture := newWebFixture(t, false, time.Minute)
	defer fixture.store.Close()
	fixture.login(t)
	fixture.app.now = func() time.Time { return fixture.now.Add(2 * time.Minute) }
	response := fixture.requestJSON(http.MethodGet, "/api/v1/admin/devices", nil, false)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("expired session status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestBackendIsAPIOnlyAndHealthEndpointWorks(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	rootResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(rootResponse, httptest.NewRequest(http.MethodGet, "/", nil))
	if rootResponse.Code != http.StatusNotFound || strings.Contains(rootResponse.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("backend unexpectedly served frontend: status=%d type=%q", rootResponse.Code, rootResponse.Header().Get("Content-Type"))
	}
	healthResponse := httptest.NewRecorder()
	fixture.app.ServeHTTP(healthResponse, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthResponse.Code != http.StatusOK || !strings.Contains(healthResponse.Body.String(), `"status":"ok"`) {
		t.Fatalf("health status=%d body=%s", healthResponse.Code, healthResponse.Body.String())
	}
}

func TestLiveStatusUsesControlledComputerLocalDateWhileOnline(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	device, err := fixture.store.CreateDevice(context.Background(), "Different timezone", fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	var quotas [7]int64
	quotas[time.Sunday] = 600
	quotas[time.Monday] = 3600
	if err := fixture.store.SaveQuotas(context.Background(), device.ID, quotas, 5, fixture.now); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.store.ReceiveHeartbeat(context.Background(), device.ID, protocol.HeartbeatRequest{
		PolicyRevision: 2, LocalDate: "2026-08-09", SecondsUsed: 1,
	}, fixture.now); err != nil {
		t.Fatal(err)
	}

	_, _, status, err := fixture.app.loadDeviceLiveStatus(context.Background(), device.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.LocalDate != "2026-08-09" || status.TodayQuotaSeconds != 600 || status.UsedSeconds != 1 {
		t.Fatalf("status did not use client date: %+v", status)
	}
	if status.GraphicalSessionActive || status.Counting {
		t.Fatalf("heartbeat without a graphical session was treated as active: %+v", status)
	}
	if _, err := fixture.store.ReceiveHeartbeat(context.Background(), device.ID, protocol.HeartbeatRequest{
		PolicyRevision: 2, LocalDate: "2026-08-09", SecondsUsed: 1,
		GraphicalSessionActive: true, GraphicalSessionID: "session-9",
		RequestSessionState: true,
	}, fixture.now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	_, _, status, err = fixture.app.loadDeviceLiveStatus(context.Background(), device.ID)
	if err != nil || !status.GraphicalSessionActive || !status.Counting {
		t.Fatalf("active graphical session not reflected in live status: %+v err=%v", status, err)
	}
}

func TestHeartbeatRequiresDeviceCredential(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	device, err := fixture.store.CreateDevice(context.Background(), "Zorin", fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	token, err := fixture.store.IssueDeviceToken(context.Background(), device.ID, fixture.now)
	if err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(protocol.HeartbeatRequest{LocalDate: fixture.now.Format("2006-01-02")})
	request := httptest.NewRequest(http.MethodPost, protocol.HeartbeatPath, bytes.NewReader(payload))
	request.Header.Set(deviceIDHeader, device.ID)
	request.Header.Set("Authorization", "Bearer wrong")
	response := httptest.NewRecorder()
	fixture.app.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("wrong token status=%d body=%s", response.Code, response.Body.String())
	}
	request = httptest.NewRequest(http.MethodPost, protocol.HeartbeatPath, bytes.NewReader(payload))
	request.Header.Set(deviceIDHeader, device.ID)
	request.Header.Set("Authorization", "Bearer "+token)
	response = httptest.NewRecorder()
	fixture.app.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("valid token status=%d body=%s", response.Code, response.Body.String())
	}
}

type testContext interface {
	Helper()
	TempDir() string
	Fatal(...interface{})
}

func newWebFixture(t testContext, secureCookies bool, sessionLifetime time.Duration) *webFixture {
	t.Helper()
	ctx := context.Background()
	store, err := serverstorage.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	parameters := localauth.Argon2Params{
		Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16,
	}
	passwordHash, err := localauth.HashPassword("secret", parameters)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.Local)
	if _, err := store.BootstrapAdmin(ctx, "admin", passwordHash, now); err != nil {
		t.Fatal(err)
	}
	application, err := New(store, secureCookies, sessionLifetime, 60*time.Second, "")
	if err != nil {
		t.Fatal(err)
	}
	application.now = func() time.Time { return now }
	return &webFixture{app: application, store: store, now: now}
}

func (f *webFixture) login(t *testing.T) {
	t.Helper()
	bootstrapResponse := f.requestJSON(http.MethodGet, "/api/v1/admin/session", nil, false)
	var anonymousSession adminSessionResponse
	decodeResponse(t, bootstrapResponse, &anonymousSession)
	loginCSRFTokenCookie := findCookie(t, bootstrapResponse.Result().Cookies(), loginCSRFCookie)

	loginBody, _ := json.Marshal(adminLoginRequest{
		Login: "admin", Password: "secret", CSRFToken: anonymousSession.CSRFToken,
	})
	loginRequest := httptest.NewRequest(http.MethodPost, "/api/v1/admin/session", bytes.NewReader(loginBody))
	loginRequest.Header.Set("Content-Type", "application/json")
	loginRequest.AddCookie(loginCSRFTokenCookie)
	loginResponse := httptest.NewRecorder()
	f.app.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginResponse.Code, loginResponse.Body.String())
	}
	f.sessionCookie = findCookie(t, loginResponse.Result().Cookies(), sessionCookieName)
	var authenticatedSession adminSessionResponse
	decodeResponse(t, loginResponse, &authenticatedSession)
	f.csrfToken = authenticatedSession.CSRFToken
}

func (f *webFixture) requestJSON(method, path string, payload interface{}, includeCSRF bool) *httptest.ResponseRecorder {
	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		encoded, _ := json.Marshal(payload)
		body = bytes.NewReader(encoded)
	}
	request := httptest.NewRequest(method, path, body)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if includeCSRF {
		request.Header.Set(csrfHeaderName, f.csrfToken)
	}
	if f.sessionCookie != nil {
		request.AddCookie(f.sessionCookie)
	}
	response := httptest.NewRecorder()
	f.app.ServeHTTP(response, request)
	return response
}

func decodeResponse(t *testing.T, response *httptest.ResponseRecorder, destination interface{}) {
	t.Helper()
	if err := json.Unmarshal(response.Body.Bytes(), destination); err != nil {
		t.Fatalf("decode response status=%d body=%s err=%v", response.Code, response.Body.String(), err)
	}
}

func findCookie(t *testing.T, cookies []*http.Cookie, name string) *http.Cookie {
	t.Helper()
	for _, cookie := range cookies {
		if cookie.Name == name && cookie.MaxAge >= 0 {
			return cookie
		}
	}
	t.Fatalf("cookie %s not found in %+v", name, cookies)
	return nil
}
