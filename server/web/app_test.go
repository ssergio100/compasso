package web

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sergio/compasso/agent/localauth"
	protocol "github.com/sergio/compasso/protocol/v1"
	serverstorage "github.com/sergio/compasso/server/storage"
)

type webFixture struct {
	app     *App
	store   *serverstorage.Store
	now     time.Time
	session *http.Cookie
	csrf    string
}

func TestLoginCorrectIncorrectAndSecureCookie(t *testing.T) {
	fixture := newWebFixture(t, true, time.Hour)
	defer fixture.store.Close()
	loginCSRF := getLoginCSRF(t, fixture.app)

	wrong := postForm(fixture.app, "/login", url.Values{
		"csrf_token": {loginCSRF.Value}, "login": {"admin"}, "password": {"wrong"},
	}, loginCSRF)
	if wrong.Code != http.StatusUnauthorized || !strings.Contains(wrong.Body.String(), "Usuário ou senha inválidos") {
		t.Fatalf("wrong login status=%d body=%s", wrong.Code, wrong.Body.String())
	}

	correct := postForm(fixture.app, "/login", url.Values{
		"csrf_token": {loginCSRF.Value}, "login": {"admin"}, "password": {"secret"},
	}, loginCSRF)
	if correct.Code != http.StatusSeeOther {
		t.Fatalf("correct login status=%d body=%s", correct.Code, correct.Body.String())
	}
	cookie := findCookie(t, correct.Result().Cookies(), sessionCookieName)
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("unsafe session cookie: %+v", cookie)
	}
	if correct.Header().Get("Strict-Transport-Security") == "" {
		t.Fatal("production HTTPS response is missing HSTS")
	}
}

func TestAdministrativeWorkflowAndCSRF(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	fixture.login(t)

	withoutCSRF := postForm(fixture.app, "/devices", url.Values{"name": {"PC"}}, fixture.session)
	if withoutCSRF.Code != http.StatusForbidden {
		t.Fatalf("missing CSRF status=%d, want 403", withoutCSRF.Code)
	}

	created := fixture.post("/devices", url.Values{"name": {"PC do quarto"}})
	if created.Code != http.StatusSeeOther {
		t.Fatalf("create device status=%d body=%s", created.Code, created.Body.String())
	}
	location := created.Header().Get("Location")
	deviceID := strings.Split(strings.TrimPrefix(location, "/devices/"), "?")[0]
	if deviceID == "" {
		t.Fatalf("create location=%q", location)
	}

	quotaValues := url.Values{"warning_minutes": {"10"}}
	for day := 0; day < 7; day++ {
		quotaValues.Set("quota_"+string(rune('0'+day)), "00:00")
	}
	quotaValues.Set("quota_1", "08:00")
	quotaValues.Set("quota_2", "00:45")
	if response := fixture.post("/devices/"+deviceID+"/quotas", quotaValues); response.Code != http.StatusSeeOther {
		t.Fatalf("save quotas status=%d body=%s", response.Code, response.Body.String())
	}
	_, storedPolicy, err := fixture.store.LoadDevice(context.Background(), deviceID)
	if err != nil || storedPolicy.WeeklyQuota[time.Monday] != 28800 || storedPolicy.WeeklyQuota[time.Tuesday] != 2700 {
		t.Fatalf("independent quotas=%v err=%v", storedPolicy.WeeklyQuota, err)
	}
	if response := fixture.post("/devices/"+deviceID+"/bonus", url.Values{"minutes": {"10"}}); response.Code != http.StatusSeeOther {
		t.Fatalf("add bonus status=%d body=%s", response.Code, response.Body.String())
	}
	allowancePage := fixture.get("/devices/" + deviceID)
	if allowancePage.Code != http.StatusOK || !strings.Contains(allowancePage.Body.String(), "Cota de hoje") ||
		!strings.Contains(allowancePage.Body.String(), "<strong>08:00</strong>") ||
		!strings.Contains(allowancePage.Body.String(), "data-remaining-seconds=\"29400\"") {
		t.Fatalf("page did not keep quota fixed while adding extra time to remaining: status=%d", allowancePage.Code)
	}
	statusResponse := fixture.get("/api/v1/admin/devices/" + deviceID + "/status")
	var liveStatus deviceLiveStatus
	if statusResponse.Code != http.StatusOK {
		t.Fatalf("live status code=%d body=%s", statusResponse.Code, statusResponse.Body.String())
	}
	if err := json.Unmarshal(statusResponse.Body.Bytes(), &liveStatus); err != nil {
		t.Fatal(err)
	}
	if liveStatus.TodayQuotaSeconds != 8*60*60 || liveStatus.RemainingSeconds != 8*60*60+10*60 {
		t.Fatalf("unexpected live status: %+v", liveStatus)
	}
	unauthenticatedStatus := httptest.NewRecorder()
	fixture.app.ServeHTTP(
		unauthenticatedStatus,
		httptest.NewRequest(http.MethodGet, "/api/v1/admin/devices/"+deviceID+"/status", nil),
	)
	if unauthenticatedStatus.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated live status code=%d, want 401", unauthenticatedStatus.Code)
	}

	routineValues := url.Values{
		"name": {"Dormir"}, "start": {"22:00"}, "end": {"08:00"},
	}
	for day := 1; day <= 5; day++ {
		routineValues.Set("day_"+string(rune('0'+day)), "on")
	}
	if response := fixture.post("/devices/"+deviceID+"/routines", routineValues); response.Code != http.StatusSeeOther {
		t.Fatalf("save routine status=%d body=%s", response.Code, response.Body.String())
	}
	_, storedPolicy, err = fixture.store.LoadDevice(context.Background(), deviceID)
	if err != nil || len(storedPolicy.Routines) != 1 || storedPolicy.Routines[0].Start != 79200 || storedPolicy.Routines[0].End != 28800 || storedPolicy.Routines[0].Days[time.Saturday] {
		t.Fatalf("stored overnight routine=%+v err=%v", storedPolicy.Routines, err)
	}

	password := "new responsible secret"
	if response := fixture.post("/devices/"+deviceID+"/password", url.Values{
		"password": {password}, "password_confirmation": {password},
	}); response.Code != http.StatusSeeOther {
		t.Fatalf("password update status=%d body=%s", response.Code, response.Body.String())
	}
	page := fixture.get("/devices/" + deviceID)
	if page.Code != http.StatusOK || strings.Contains(page.Body.String(), password) || strings.Contains(page.Body.String(), "$argon2id$") {
		t.Fatalf("device page leaked password or verifier: status=%d", page.Code)
	}
	events, err := fixture.store.ListAudit(context.Background(), deviceID, 20)
	if err != nil || len(events) < 4 {
		t.Fatalf("audit events=%+v err=%v", events, err)
	}
}

func TestExpiredSessionRedirectsToLogin(t *testing.T) {
	fixture := newWebFixture(t, false, time.Minute)
	defer fixture.store.Close()
	fixture.login(t)
	fixture.app.now = func() time.Time { return fixture.now.Add(2 * time.Minute) }
	response := fixture.get("/devices")
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("expired session status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
}

func TestExternalTemplateChangesAppearWithoutRestart(t *testing.T) {
	fixture := newWebFixture(t, false, time.Hour)
	defer fixture.store.Close()
	assetsDirectory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(assetsDirectory, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, templateName := range []string{"base.html", "login.html", "devices.html", "device.html"} {
		templateContents, err := os.ReadFile(filepath.Join("templates", templateName))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(assetsDirectory, "templates", templateName), templateContents, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	application, err := NewWithAssets(fixture.store, false, time.Hour, assetsDirectory)
	if err != nil {
		t.Fatal(err)
	}
	loginTemplatePath := filepath.Join(assetsDirectory, "templates", "login.html")
	loginTemplate, err := os.ReadFile(loginTemplatePath)
	if err != nil {
		t.Fatal(err)
	}
	updatedLoginTemplate := strings.Replace(string(loginTemplate), "Entrar", "Acesso atualizado", 1)
	if err := os.WriteFile(loginTemplatePath, []byte(updatedLoginTemplate), 0o644); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	application.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Acesso atualizado") {
		t.Fatalf("external template was not reloaded: status=%d body=%s", response.Code, response.Body.String())
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

func newWebFixture(t testContext, secure bool, lifetime time.Duration) *webFixture {
	t.Helper()
	ctx := context.Background()
	store, err := serverstorage.Open(ctx, filepath.Join(t.TempDir(), "server.db"))
	if err != nil {
		t.Fatal(err)
	}
	params := localauth.Argon2Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 8, KeyLength: 16}
	hash, err := localauth.HashPassword("secret", params)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, time.August, 10, 12, 0, 0, 0, time.Local)
	if _, err := store.BootstrapAdmin(ctx, "admin", hash, now); err != nil {
		t.Fatal(err)
	}
	app, err := NewWithAssets(store, secure, lifetime, ".")
	if err != nil {
		t.Fatal(err)
	}
	app.now = func() time.Time { return now }
	return &webFixture{app: app, store: store, now: now}
}

func (f *webFixture) login(t *testing.T) {
	t.Helper()
	loginCSRF := getLoginCSRF(t, f.app)
	response := postForm(f.app, "/login", url.Values{
		"csrf_token": {loginCSRF.Value}, "login": {"admin"}, "password": {"secret"},
	}, loginCSRF)
	f.session = findCookie(t, response.Result().Cookies(), sessionCookieName)
	value, ok := f.app.sessions.get(f.session.Value, f.now)
	if !ok {
		t.Fatal("created session not found")
	}
	f.csrf = value.CSRF
}

func (f *webFixture) post(path string, values url.Values) *httptest.ResponseRecorder {
	values.Set("csrf_token", f.csrf)
	return postForm(f.app, path, values, f.session)
}

func (f *webFixture) get(path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	if f.session != nil {
		request.AddCookie(f.session)
	}
	response := httptest.NewRecorder()
	f.app.ServeHTTP(response, request)
	return response
}

func getLoginCSRF(t *testing.T, app *App) *http.Cookie {
	t.Helper()
	response := httptest.NewRecorder()
	app.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/login", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("login page status=%d body=%s", response.Code, response.Body.String())
	}
	return findCookie(t, response.Result().Cookies(), loginCSRFCookie)
}

func postForm(handler http.Handler, path string, values url.Values, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for _, cookie := range cookies {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
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
