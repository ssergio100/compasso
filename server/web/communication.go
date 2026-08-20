package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ssergio100/compasso/server/storage"
)

type statusCapturingResponseWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusCapturingResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusCapturingResponseWriter) Write(body []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(body)
}

func (w *statusCapturingResponseWriter) statusCode() int {
	if w.status == 0 {
		return http.StatusOK
	}
	return w.status
}

func (a *App) logAdministrativeCommunication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deviceID, operation, route, shouldLog := administrativeCommunication(r)
		if !shouldLog {
			next.ServeHTTP(w, r)
			return
		}
		started := time.Now()
		correlationID, _ := randomToken()
		if correlationID != "" {
			w.Header().Set("X-Compasso-Correlation-ID", correlationID)
		}
		recorder := &statusCapturingResponseWriter{ResponseWriter: w}
		next.ServeHTTP(recorder, r)
		status := recorder.statusCode()
		_, _ = a.store.AppendCommunicationLog(r.Context(), storage.CommunicationLog{
			DeviceID: deviceID, Source: "interface", Target: "api", Operation: operation,
			Result: communicationResultForStatus(status), HTTPStatus: status,
			DurationMS: elapsedMilliseconds(started), Summary: administrativeCommunicationSummary(operation, status),
			Details: map[string]string{
				"correlation_id": correlationID,
				"method":         r.Method,
				"route":          route,
			},
		}, a.now())
	})
}

func administrativeCommunication(r *http.Request) (deviceID, operation, route string, ok bool) {
	const prefix = "/api/v1/admin/devices/"
	if r.Method == http.MethodOptions || !strings.HasPrefix(r.URL.Path, prefix) {
		return "", "", "", false
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, prefix), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		return "", "", "", false
	}
	deviceID = parts[0]
	resource := "device"
	if len(parts) > 1 {
		resource = parts[1]
	}
	if resource == "communication" {
		return "", "", "", false
	}
	operation = r.Method + " " + resource
	route = prefix + "{device_id}"
	if resource != "device" {
		route += "/" + resource
	}
	if len(parts) > 2 {
		route += "/{resource_id}"
	}
	return deviceID, operation, route, true
}

func administrativeCommunicationSummary(operation string, status int) string {
	if status >= 400 {
		return "Solicitação da interface rejeitada pela API."
	}
	return "Solicitação da interface processada pela API: " + operation + "."
}

func communicationResultForStatus(status int) string {
	if status >= 500 {
		return "error"
	}
	if status >= 400 {
		return "warning"
	}
	return "success"
}

func elapsedMilliseconds(started time.Time) int64 {
	elapsed := time.Since(started).Milliseconds()
	if elapsed < 1 {
		return 1
	}
	return elapsed
}

func (a *App) adminDeviceCommunicationAPI(
	w http.ResponseWriter,
	r *http.Request,
	current session,
	deviceID string,
	pathParts []string,
) {
	if len(pathParts) == 1 && pathParts[0] == "settings" {
		a.adminCommunicationSettingsAPI(w, r, current)
		return
	}
	if len(pathParts) != 0 {
		writeJSONError(w, http.StatusNotFound, "not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		limit, afterID, ok := communicationQuery(w, r)
		if !ok {
			return
		}
		events, err := a.store.ListCommunicationLogs(r.Context(), deviceID, afterID, limit)
		if !writeAdminReadError(w, err) {
			return
		}
		retentionDays, err := a.store.CommunicationRetentionDays(r.Context())
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not load communication settings")
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"events": events, "retention_days": retentionDays,
		})
	case http.MethodDelete:
		if !requireAdminCSRF(w, r, current) {
			return
		}
		deleted, err := a.store.DeleteCommunicationLogs(r.Context(), deviceID)
		if !writeAdminReadError(w, err) {
			return
		}
		writeJSON(w, http.StatusOK, map[string]int64{"deleted": deleted})
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (a *App) adminCommunicationSettingsAPI(w http.ResponseWriter, r *http.Request, current session) {
	if r.Method != http.MethodPut {
		w.Header().Set("Allow", http.MethodPut)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if !requireAdminCSRF(w, r, current) {
		return
	}
	var request updateCommunicationRetentionRequest
	if err := decodeJSONBody(w, r, &request); err != nil || request.RetentionDays < 1 || request.RetentionDays > 365 {
		writeJSONError(w, http.StatusBadRequest, "retention must be between 1 and 365 days")
		return
	}
	if err := a.store.SetCommunicationRetentionDays(r.Context(), request.RetentionDays, a.now()); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid communication retention")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"retention_days": request.RetentionDays})
}

func communicationQuery(w http.ResponseWriter, r *http.Request) (limit int, afterID int64, ok bool) {
	limit = 200
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 500 {
			writeJSONError(w, http.StatusBadRequest, "limit must be between 1 and 500")
			return 0, 0, false
		}
		limit = parsed
	}
	if value := r.URL.Query().Get("after"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 0 {
			writeJSONError(w, http.StatusBadRequest, "after must be a non-negative integer")
			return 0, 0, false
		}
		afterID = parsed
	}
	return limit, afterID, true
}
