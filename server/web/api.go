package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	protocol "github.com/ssergio100/compasso/protocol/v1"
	"github.com/ssergio100/compasso/server/storage"
)

const deviceIDHeader = "X-Tempo-Device-ID"

func (a *App) heartbeat(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	deviceID := strings.TrimSpace(r.Header.Get(deviceIDHeader))
	correlationID, _ := randomToken()
	if correlationID != "" {
		w.Header().Set("X-Compasso-Correlation-ID", correlationID)
	}
	protocolVersion := strings.TrimSpace(r.Header.Get(protocol.VersionHeader))
	if protocolVersion == "" {
		protocolVersion = "1"
	}
	details := map[string]string{
		"correlation_id": correlationID, "protocol_version": protocolVersion,
		"route": protocol.HeartbeatPath,
	}
	recorder := &statusCapturingResponseWriter{ResponseWriter: w}
	a.handleHeartbeat(recorder, r, details)
	if deviceID == "" {
		return
	}
	status := recorder.statusCode()
	result := communicationResultForStatus(status)
	summary := "Heartbeat processado e resposta enviada ao agente."
	if status >= 400 {
		summary = "Heartbeat rejeitado pela API."
	}
	heartbeatLog, _ := a.store.AppendCommunicationLog(r.Context(), storage.CommunicationLog{
		DeviceID: deviceID, Source: "agent", Target: "api", Operation: "heartbeat",
		Result: result, HTTPStatus: status, DurationMS: elapsedMilliseconds(started), Summary: summary,
		Details: details,
	}, a.now())
	a.publishCommunicationLog(deviceID, heartbeatLog)
	if status == http.StatusOK && heartbeatCarriedState(details) {
		responseLog, _ := a.store.AppendCommunicationLog(r.Context(), storage.CommunicationLog{
			DeviceID: deviceID, Source: "api", Target: "agent", Operation: "heartbeat_response",
			Result: "success", HTTPStatus: status, DurationMS: elapsedMilliseconds(started),
			Summary: "API enviou atualizações ou confirmações ao agente.", Details: details,
		}, a.now())
		a.publishCommunicationLog(deviceID, responseLog)
	}
}

func (a *App) handleHeartbeat(w http.ResponseWriter, r *http.Request, details map[string]string) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	deviceID := strings.TrimSpace(r.Header.Get(deviceIDHeader))
	authorization := r.Header.Get("Authorization")
	ok := strings.HasPrefix(authorization, "Bearer ")
	token := strings.TrimPrefix(authorization, "Bearer ")
	if !ok || strings.ContainsAny(token, " \t\r\n") || a.store.AuthenticateDevice(r.Context(), deviceID, token) != nil {
		w.Header().Set("WWW-Authenticate", "Bearer")
		writeJSONError(w, http.StatusUnauthorized, "invalid device credentials")
		return
	}
	protocolVersion := strings.TrimSpace(r.Header.Get(protocol.VersionHeader))
	if protocolVersion == "" {
		protocolVersion = "1"
	}
	if protocolVersion != "1" && protocolVersion != protocol.CurrentProtocolVersion {
		writeJSONError(w, http.StatusBadRequest, "unsupported protocol version")
		return
	}
	if protocolVersion == "1" {
		pendingBonus, err := a.store.HasPendingRemoteBonus(r.Context(), deviceID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "could not inspect pending operations")
			return
		}
		if pendingBonus {
			writeJSONErrorResponse(w, http.StatusUpgradeRequired, protocol.ErrorResponse{
				Error: "agent update required before applying pending bonus", Code: "agent_upgrade_required",
			})
			return
		}
	}
	r.Body = http.MaxBytesReader(w, r.Body, 256<<10)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var request protocol.HeartbeatRequest
	if err := decoder.Decode(&request); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid heartbeat payload")
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeJSONError(w, http.StatusBadRequest, "invalid heartbeat payload")
		return
	}
	details["local_date"] = request.LocalDate
	details["policy_revision"] = strconv.FormatInt(request.PolicyRevision, 10)
	details["control_revision"] = strconv.FormatInt(request.ControlRevision, 10)
	details["session_state_revision"] = strconv.FormatInt(request.SessionStateRevision, 10)
	details["used_seconds"] = strconv.FormatInt(request.SecondsUsed, 10)
	details["request_events"] = strconv.Itoa(len(request.Events))
	details["command_acknowledgements"] = strconv.Itoa(len(request.CommandAcks))
	response, err := a.store.ReceiveHeartbeat(r.Context(), deviceID, request, a.now())
	if err != nil {
		status := http.StatusBadRequest
		var revisionError *storage.RevisionAheadError
		if errors.As(err, &revisionError) {
			status = http.StatusConflict
			writeJSONErrorResponse(w, status, protocol.ErrorResponse{
				Error:          "client synchronization state is newer than this device",
				Code:           "revision_ahead",
				ClientRevision: revisionError.ClientRevision,
				ServerRevision: revisionError.ServerRevision,
			})
			return
		}
		writeJSONError(w, status, "heartbeat rejected")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	details["policy_sent"] = strconv.FormatBool(response.Policy != nil)
	details["session_state_sent"] = strconv.FormatBool(response.SessionState != nil)
	details["response_commands"] = strconv.Itoa(len(response.Commands))
	details["acknowledged_events"] = strconv.Itoa(len(response.AcknowledgedEvents))
	a.publishDeviceStatus(deviceID, "status")
	_ = json.NewEncoder(w).Encode(response)
}

func heartbeatCarriedState(details map[string]string) bool {
	return details["policy_sent"] == "true" || details["session_state_sent"] == "true" ||
		details["response_commands"] != "0" || details["acknowledged_events"] != "0"
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSONErrorResponse(w, status, protocol.ErrorResponse{Error: message})
}

func writeJSONErrorResponse(w http.ResponseWriter, status int, response protocol.ErrorResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}
