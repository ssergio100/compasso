package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ssergio100/compasso/agent/localauth"
	"github.com/ssergio100/compasso/server/storage"
)

const csrfHeaderName = "X-CSRF-Token"

type adminSessionResponse struct {
	Authenticated bool   `json:"authenticated"`
	Login         string `json:"login,omitempty"`
	CSRFToken     string `json:"csrf_token"`
	SetupRequired bool   `json:"setup_required"`
}

type adminLoginRequest struct {
	Login     string `json:"login"`
	Password  string `json:"password"`
	CSRFToken string `json:"csrf_token"`
}

type adminBonusResponse struct {
	Message     string `json:"message"`
	OperationID string `json:"operation_id"`
}

type adminBonusStatusResponse struct {
	Acknowledged bool `json:"acknowledged"`
}

type updateCommunicationRetentionRequest struct {
	RetentionDays int `json:"retention_days"`
}

type adminSetupRequest struct {
	Login                string `json:"login"`
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
	CSRFToken            string `json:"csrf_token"`
}

type adminDeviceResponse struct {
	ID                     string     `json:"id"`
	Name                   string     `json:"name"`
	AvatarKey              string     `json:"avatar_key"`
	LastSeenAt             *time.Time `json:"last_seen_at"`
	PolicyRevision         int64      `json:"policy_revision"`
	AppliedPolicyRevision  int64      `json:"applied_policy_revision"`
	GraphicalSessionActive bool       `json:"graphical_session_active"`
	Online                 bool       `json:"online"`
}

type adminRoutineResponse struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	IconKey string  `json:"icon_key"`
	Days    [7]bool `json:"days"`
	Start   int64   `json:"start_second"`
	End     int64   `json:"end_second"`
	Enabled bool    `json:"enabled"`
}

type adminPolicyResponse struct {
	Revision         int64                  `json:"revision"`
	MonitoringPaused bool                   `json:"monitoring_paused"`
	ManualBlock      bool                   `json:"manual_block"`
	WarningMinutes   int                    `json:"warning_minutes"`
	PasswordSet      bool                   `json:"password_set"`
	WeeklyQuota      [7]int64               `json:"weekly_quota_seconds"`
	Routines         []adminRoutineResponse `json:"routines"`
}

type adminAuditEventResponse struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Origin    string    `json:"origin"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}

type adminDeviceDetailResponse struct {
	Device  adminDeviceResponse       `json:"device"`
	Policy  adminPolicyResponse       `json:"policy"`
	Control adminControlResponse      `json:"control"`
	Status  deviceLiveStatus          `json:"status"`
	Events  []adminAuditEventResponse `json:"events"`
}

type adminControlResponse struct {
	Revision         int64 `json:"revision"`
	MonitoringPaused bool  `json:"monitoring_paused"`
	ManualBlock      bool  `json:"manual_block"`
}

type createAdminDeviceRequest struct {
	Name      string `json:"name"`
	AvatarKey string `json:"avatar_key"`
}

type updateAdminDeviceRequest struct {
	Name      string `json:"name"`
	AvatarKey string `json:"avatar_key"`
}

type updateAdminPolicyRequest struct {
	WeeklyQuota    [7]int64 `json:"weekly_quota_seconds"`
	WarningMinutes int      `json:"warning_minutes"`
}

type saveAdminRoutineRequest struct {
	Name    string  `json:"name"`
	IconKey string  `json:"icon_key"`
	Days    [7]bool `json:"days"`
	Start   int64   `json:"start_second"`
	End     int64   `json:"end_second"`
	Enabled bool    `json:"enabled"`
}

type updateAdminPasswordRequest struct {
	Password             string `json:"password"`
	PasswordConfirmation string `json:"password_confirmation"`
}

type addAdminBonusRequest struct {
	Minutes int `json:"minutes"`
}

type queueAdminCommandRequest struct {
	Command string `json:"command"`
}

func (a *App) corsHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/v1/admin/") {
			next.ServeHTTP(w, r)
			return
		}
		origin := strings.TrimRight(strings.TrimSpace(r.Header.Get("Origin")), "/")
		if origin != "" {
			allowedOrigin, allowed := a.allowedAdminOrigin(origin, r.Host)
			if !allowed {
				writeJSONError(w, http.StatusForbidden, "origin not allowed")
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token")
			w.Header().Set("Access-Control-Max-Age", "600")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) adminSessionAPI(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		if current, _, authenticated := a.authenticated(r); authenticated {
			writeJSON(w, http.StatusOK, adminSessionResponse{
				Authenticated: true, Login: current.Login, CSRFToken: current.CSRF,
			})
			return
		}
		csrfToken, err := randomToken()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		hasAdministrators, err := a.store.HasAdministrators(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not inspect initial setup")
			return
		}
		a.setCookie(w, &http.Cookie{Name: loginCSRFCookie, Value: csrfToken, HttpOnly: true, MaxAge: 600})
		writeJSON(w, http.StatusOK, adminSessionResponse{CSRFToken: csrfToken, SetupRequired: !hasAdministrators})
	case http.MethodPost:
		var request adminLoginRequest
		if err := decodeJSONBody(w, r, &request); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		csrfCookie, err := r.Cookie(loginCSRFCookie)
		if err != nil || !constantEqual(csrfCookie.Value, request.CSRFToken) {
			writeJSONError(w, http.StatusForbidden, "invalid CSRF token")
			return
		}
		admin, err := a.store.AdminByLogin(r.Context(), request.Login)
		validCredentials := false
		if err == nil && admin.Active {
			validCredentials, _ = localauth.VerifyPassword(request.Password, admin.PasswordHash)
		}
		if !validCredentials {
			writeJSONError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		sessionToken, current, err := a.sessions.create(admin.ID, admin.Login, a.now())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "internal error")
			return
		}
		a.setCookie(w, &http.Cookie{
			Name: sessionCookieName, Value: sessionToken, HttpOnly: true,
			Expires: current.Expires, MaxAge: int(a.sessions.lifetime.Seconds()),
		})
		a.setCookie(w, &http.Cookie{Name: loginCSRFCookie, MaxAge: -1, HttpOnly: true})
		writeJSON(w, http.StatusOK, adminSessionResponse{
			Authenticated: true, Login: current.Login, CSRFToken: current.CSRF,
		})
	case http.MethodDelete:
		current, sessionToken, authenticated := a.requireAdminAPISession(w, r)
		if !authenticated {
			return
		}
		if !constantEqual(r.Header.Get(csrfHeaderName), current.CSRF) {
			writeJSONError(w, http.StatusForbidden, "invalid CSRF token")
			return
		}
		a.sessions.delete(sessionToken)
		a.setCookie(w, &http.Cookie{Name: sessionCookieName, MaxAge: -1, HttpOnly: true})
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *App) adminSetupAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request adminSetupRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	csrfCookie, err := r.Cookie(loginCSRFCookie)
	if err != nil || !constantEqual(csrfCookie.Value, request.CSRFToken) {
		writeJSONError(w, http.StatusForbidden, "invalid CSRF token")
		return
	}
	hasAdministrators, err := a.store.HasAdministrators(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not inspect initial setup")
		return
	}
	if hasAdministrators {
		writeJSONError(w, http.StatusConflict, "initial setup already completed")
		return
	}
	login := strings.TrimSpace(request.Login)
	if login == "" || len(login) > 80 || request.Password == "" || len(request.Password) > 4096 || request.Password != request.PasswordConfirmation {
		writeJSONError(w, http.StatusBadRequest, "invalid initial administrator")
		return
	}
	passwordHash, err := localauth.HashPassword(request.Password, localauth.DefaultArgon2Params)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not secure administrator password")
		return
	}
	created, err := a.store.BootstrapAdmin(r.Context(), login, passwordHash, a.now())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create initial administrator")
		return
	}
	if !created {
		writeJSONError(w, http.StatusConflict, "initial setup already completed")
		return
	}
	administrator, err := a.store.AdminByLogin(r.Context(), login)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not start administrator session")
		return
	}
	sessionToken, current, err := a.sessions.create(administrator.ID, administrator.Login, a.now())
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not start administrator session")
		return
	}
	a.setCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: sessionToken, HttpOnly: true,
		Expires: current.Expires, MaxAge: int(a.sessions.lifetime.Seconds()),
	})
	a.setCookie(w, &http.Cookie{Name: loginCSRFCookie, MaxAge: -1, HttpOnly: true})
	writeJSON(w, http.StatusCreated, adminSessionResponse{
		Authenticated: true, Login: current.Login, CSRFToken: current.CSRF,
	})
}

func (a *App) adminDevicesAPI(w http.ResponseWriter, r *http.Request) {
	current, _, authenticated := a.requireAdminAPISession(w, r)
	if !authenticated {
		return
	}
	switch r.Method {
	case http.MethodGet:
		devices, err := a.store.ListDevices(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not load devices")
			return
		}
		response := make([]adminDeviceResponse, 0, len(devices))
		for _, device := range devices {
			response = append(response, a.adminDeviceResponse(device))
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"devices": response})
	case http.MethodPost:
		if !requireAdminCSRF(w, r, current) {
			return
		}
		var request createAdminDeviceRequest
		if err := decodeJSONBody(w, r, &request); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		device, err := a.store.CreateDeviceWithAvatar(r.Context(), request.Name, request.AvatarKey, a.now())
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid device")
			return
		}
		writeJSON(w, http.StatusCreated, a.adminDeviceResponse(device))
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *App) adminDeviceAPI(w http.ResponseWriter, r *http.Request) {
	current, _, authenticated := a.requireAdminAPISession(w, r)
	if !authenticated {
		return
	}
	pathParts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/admin/devices/"), "/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	deviceID := pathParts[0]
	if len(pathParts) == 1 {
		a.adminDeviceRootAPI(w, r, current, deviceID)
		return
	}
	if len(pathParts) > 2 && pathParts[1] != "routines" && pathParts[1] != "commands" && pathParts[1] != "communication" && pathParts[1] != "activities" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	switch pathParts[1] {
	case "status":
		a.adminDeviceStatusAPI(w, r, deviceID)
	case "stream":
		a.adminDeviceStreamAPI(w, r, deviceID)
	case "policy":
		a.adminDevicePolicyAPI(w, r, current, deviceID)
	case "routines":
		a.adminDeviceRoutinesAPI(w, r, current, deviceID, pathParts[2:])
	case "password":
		a.adminDevicePasswordAPI(w, r, current, deviceID)
	case "token":
		a.adminDeviceTokenAPI(w, r, current, deviceID)
	case "bonus":
		a.adminDeviceBonusAPI(w, r, current, deviceID)
	case "commands":
		if len(pathParts) == 3 {
			a.adminDeviceBonusStatusAPI(w, r, deviceID, pathParts[2])
		} else if len(pathParts) == 2 {
			a.adminDeviceCommandAPI(w, r, current, deviceID)
		} else {
			writeJSONError(w, http.StatusNotFound, "not found")
		}
	case "events":
		a.adminDeviceEventsAPI(w, r, deviceID)
	case "activities":
		a.adminDeviceActivitiesAPI(w, r, current, deviceID, pathParts[2:])
	case "communication":
		a.adminDeviceCommunicationAPI(w, r, current, deviceID, pathParts[2:])
	default:
		writeJSONError(w, http.StatusNotFound, "not found")
	}
}

func (a *App) adminDeviceRootAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string) {
	switch r.Method {
	case http.MethodGet:
		device, storedPolicy, liveStatus, err := a.loadDeviceLiveStatus(r.Context(), deviceID)
		if !writeAdminReadError(w, err) {
			return
		}
		control, err := a.store.LoadControl(r.Context(), deviceID)
		if !writeAdminReadError(w, err) {
			return
		}
		events, err := a.store.ListAudit(r.Context(), deviceID, 30)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not load audit events")
			return
		}
		writeJSON(w, http.StatusOK, adminDeviceDetailResponse{
			Device: a.adminDeviceResponse(device), Policy: adminPolicyResponseFromStorage(storedPolicy),
			Control: adminControlResponse{Revision: control.Revision, MonitoringPaused: control.MonitoringPaused, ManualBlock: control.ManualBlock},
			Status:  liveStatus, Events: adminAuditEventsResponse(events),
		})
	case http.MethodPatch:
		if !requireAdminCSRF(w, r, current) {
			return
		}
		var request updateAdminDeviceRequest
		if err := decodeJSONBody(w, r, &request); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request")
			return
		}
		var err error
		if request.AvatarKey == "" {
			err = a.store.RenameDevice(r.Context(), deviceID, request.Name, a.now())
		} else {
			err = a.store.UpdateDeviceIdentity(r.Context(), deviceID, request.Name, request.AvatarKey, a.now())
		}
		if err != nil {
			writeAdminMutationError(w, err)
			return
		}
		addCommunicationDetail(r, "device_name", request.Name)
		if request.AvatarKey != "" {
			addCommunicationDetail(r, "avatar_key", request.AvatarKey)
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "device identity updated"})
	case http.MethodDelete:
		if !requireAdminCSRF(w, r, current) {
			return
		}
		if err := a.store.DeleteDevice(r.Context(), deviceID); err != nil {
			writeAdminMutationError(w, err)
			return
		}
		addCommunicationDetail(r, "action", "device_deleted")
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, PATCH, DELETE")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *App) adminDeviceStatusAPI(w http.ResponseWriter, r *http.Request, deviceID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	_, _, liveStatus, err := a.loadDeviceLiveStatus(r.Context(), deviceID)
	if !writeAdminReadError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, liveStatus)
}

func (a *App) adminDevicePolicyAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireAdminCSRF(w, r, current) {
		return
	}
	var request updateAdminPolicyRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := a.store.SaveQuotas(r.Context(), deviceID, request.WeeklyQuota, request.WarningMinutes, a.now()); err != nil {
		writeAdminMutationError(w, err)
		return
	}
	addCommunicationDetail(r, "warning_minutes", strconv.Itoa(request.WarningMinutes))
	writeJSON(w, http.StatusOK, map[string]string{"message": "policy updated"})
}

func (a *App) adminDeviceRoutinesAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string, pathParts []string) {
	if !requireAdminCSRF(w, r, current) {
		return
	}
	routineID := ""
	if len(pathParts) > 1 || (len(pathParts) == 1 && pathParts[0] == "") {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	if len(pathParts) == 1 {
		routineID = pathParts[0]
	}
	if r.Method == http.MethodDelete {
		if routineID == "" {
			writeJSONError(w, http.StatusNotFound, "not found")
			return
		}
		if err := a.store.DeleteRoutine(r.Context(), deviceID, routineID, a.now()); err != nil {
			writeAdminMutationError(w, err)
			return
		}
		addCommunicationDetail(r, "routine_action", "deleted")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		w.Header().Set("Allow", "POST, PUT, DELETE")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if r.Method == http.MethodPost && routineID != "" || r.Method == http.MethodPut && routineID == "" {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	var request saveAdminRoutineRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	createdRoutineID, err := a.store.SaveRoutine(r.Context(), deviceID, storage.Routine{
		ID: routineID, Name: request.Name, IconKey: request.IconKey, Days: request.Days, Start: request.Start,
		End: request.End, Enabled: request.Enabled,
	}, a.now())
	if err != nil {
		writeAdminMutationError(w, err)
		return
	}
	status := http.StatusOK
	if routineID == "" {
		status = http.StatusCreated
	}
	action := "updated"
	if status == http.StatusCreated {
		action = "created"
	}
	addCommunicationDetail(r, "routine_name", request.Name)
	addCommunicationDetail(r, "routine_action", action)
	writeJSON(w, status, map[string]string{"id": createdRoutineID})
}

func (a *App) adminDevicePasswordAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireAdminCSRF(w, r, current) {
		return
	}
	var request updateAdminPasswordRequest
	if err := decodeJSONBody(w, r, &request); err != nil || request.Password == "" || request.Password != request.PasswordConfirmation {
		writeJSONError(w, http.StatusBadRequest, "password and confirmation must match")
		return
	}
	verifier, err := localauth.HashPassword(request.Password, localauth.DefaultArgon2Params)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not update password")
		return
	}
	if err := a.store.SetLocalPassword(r.Context(), deviceID, verifier, a.now()); err != nil {
		writeAdminMutationError(w, err)
		return
	}
	addCommunicationDetail(r, "action", "password_updated")
	writeJSON(w, http.StatusOK, map[string]string{"message": "password updated"})
}

func (a *App) adminDeviceTokenAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		w.Header().Set("Allow", "POST, DELETE")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireAdminCSRF(w, r, current) {
		return
	}
	if r.Method == http.MethodDelete {
		if err := a.store.RevokeDeviceToken(r.Context(), deviceID, a.now()); err != nil {
			writeAdminMutationError(w, err)
			return
		}
		addCommunicationDetail(r, "action", "token_revoked")
		w.WriteHeader(http.StatusNoContent)
		return
	}
	token, err := a.store.IssueDeviceToken(r.Context(), deviceID, a.now())
	if err != nil {
		writeAdminMutationError(w, err)
		return
	}
	addCommunicationDetail(r, "action", "token_issued")
	writeJSON(w, http.StatusCreated, map[string]string{"device_id": deviceID, "device_token": token})
}

func (a *App) adminDeviceBonusAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireAdminCSRF(w, r, current) {
		return
	}
	var request addAdminBonusRequest
	if err := decodeJSONBody(w, r, &request); err != nil || request.Minutes <= 0 || request.Minutes > 12*60 {
		writeJSONError(w, http.StatusBadRequest, "bonus must be between 1 minute and 12 hours")
		return
	}
	operationID, err := a.store.QueueRemoteBonus(
		r.Context(), deviceID, int64(request.Minutes*60), a.now(),
	)
	if err != nil {
		writeAdminMutationError(w, err)
		return
	}
	addCommunicationDetail(r, "bonus_minutes", strconv.Itoa(request.Minutes))
	addCommunicationDetail(r, "operation_id", operationID)
	a.publishDeviceActivity(deviceID, operationID)
	writeJSON(w, http.StatusAccepted, adminBonusResponse{
		Message: "bonus queued", OperationID: operationID,
	})
}

func (a *App) adminDeviceActivitiesAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string, pathParts []string) {
	if r.Method == http.MethodDelete && len(pathParts) == 1 && pathParts[0] == "completed" {
		if !requireAdminCSRF(w, r, current) {
			return
		}
		deleted, err := a.store.DeleteCompletedDeviceActivities(r.Context(), deviceID)
		if !writeAdminReadError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodDelete)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(pathParts) > 1 {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	if len(pathParts) == 1 {
		activity, err := a.store.LoadDeviceActivity(r.Context(), deviceID, pathParts[0])
		if !writeAdminReadError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, activity)
		return
	}
	if _, err := a.store.CleanupExpiredCompletedActivities(r.Context(), a.now()); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not clean completed activities")
		return
	}
	limit := 100
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			writeJSONError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}
	activities, err := a.store.ListDeviceActivities(r.Context(), deviceID, limit)
	if !writeAdminReadError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"activities": activities})
}

func (a *App) adminDeviceBonusStatusAPI(w http.ResponseWriter, r *http.Request, deviceID, operationID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	acknowledged, err := a.store.RemoteBonusAcknowledged(r.Context(), deviceID, operationID)
	if !writeAdminReadError(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, adminBonusStatusResponse{Acknowledged: acknowledged})
}

func (a *App) adminDeviceCommandAPI(w http.ResponseWriter, r *http.Request, current session, deviceID string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireAdminCSRF(w, r, current) {
		return
	}
	var request queueAdminCommandRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request")
		return
	}
	operationID, err := a.store.QueueControlOperation(r.Context(), deviceID, request.Command, a.now())
	if err != nil {
		writeAdminMutationError(w, err)
		return
	}
	addCommunicationDetail(r, "command", request.Command)
	addCommunicationDetail(r, "operation_id", operationID)
	a.publishDeviceActivity(deviceID, operationID)
	writeJSON(w, http.StatusAccepted, map[string]string{"message": "command queued", "operation_id": operationID})
}

func (a *App) adminDeviceEventsAPI(w http.ResponseWriter, r *http.Request, deviceID string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 30
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 200 {
			writeJSONError(w, http.StatusBadRequest, "limit must be between 1 and 200")
			return
		}
		limit = parsed
	}
	events, err := a.store.ListAudit(r.Context(), deviceID, limit)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load audit events")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"events": adminAuditEventsResponse(events)})
}

func (a *App) requireAdminAPISession(w http.ResponseWriter, r *http.Request) (session, string, bool) {
	current, token, authenticated := a.authenticated(r)
	if !authenticated {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return session{}, "", false
	}
	return current, token, true
}

func requireAdminCSRF(w http.ResponseWriter, r *http.Request, current session) bool {
	if !constantEqual(r.Header.Get(csrfHeaderName), current.CSRF) {
		writeJSONError(w, http.StatusForbidden, "invalid CSRF token")
		return false
	}
	return true
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination interface{}) error {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain one JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeAdminReadError(w http.ResponseWriter, err error) bool {
	if err == nil {
		return true
	}
	if errors.Is(err, storage.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "device not found")
	} else {
		writeJSONError(w, http.StatusInternalServerError, "could not load device")
	}
	return false
}

func writeAdminMutationError(w http.ResponseWriter, err error) {
	if errors.Is(err, storage.ErrNotFound) {
		writeJSONError(w, http.StatusNotFound, "device or resource not found")
		return
	}
	var conflict *storage.RoutineConflictError
	if errors.As(err, &conflict) {
		writeJSONError(w, http.StatusConflict, "Este intervalo já está ocupado pela rotina “"+conflict.RoutineName+"”.")
		return
	}
	writeJSONError(w, http.StatusBadRequest, "invalid request")
}

func (a *App) adminDeviceResponse(device storage.Device) adminDeviceResponse {
	return adminDeviceResponse{
		ID: device.ID, Name: device.Name, AvatarKey: device.AvatarKey, LastSeenAt: device.LastSeenAt,
		PolicyRevision: device.PolicyRevision, AppliedPolicyRevision: device.AppliedPolicyRevision,
		GraphicalSessionActive: device.GraphicalSessionActive,
		Online:                 isOnline(device.LastSeenAt, a.now(), a.onlineTimeout),
	}
}

func adminPolicyResponseFromStorage(storedPolicy storage.Policy) adminPolicyResponse {
	routines := make([]adminRoutineResponse, 0, len(storedPolicy.Routines))
	for _, routine := range storedPolicy.Routines {
		routines = append(routines, adminRoutineResponse{
			ID: routine.ID, Name: routine.Name, IconKey: routine.IconKey, Days: routine.Days, Start: routine.Start,
			End: routine.End, Enabled: routine.Enabled,
		})
	}
	return adminPolicyResponse{
		Revision: storedPolicy.Revision, MonitoringPaused: storedPolicy.MonitoringPaused,
		ManualBlock: storedPolicy.ManualBlock, WarningMinutes: storedPolicy.WarningMinutes,
		PasswordSet: storedPolicy.LocalPasswordVerifier != "", WeeklyQuota: storedPolicy.WeeklyQuota,
		Routines: routines,
	}
}

func adminAuditEventsResponse(events []storage.AuditEvent) []adminAuditEventResponse {
	response := make([]adminAuditEventResponse, 0, len(events))
	for _, event := range events {
		response = append(response, adminAuditEventResponse{
			ID: event.ID, Kind: event.Kind, Origin: event.Origin,
			Details: event.Details, CreatedAt: event.CreatedAt,
		})
	}
	return response
}
