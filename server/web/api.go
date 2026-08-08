package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	protocol "github.com/sergio/compasso/protocol/v1"
	"github.com/sergio/compasso/server/storage"
)

const deviceIDHeader = "X-Tempo-Device-ID"

func (a *App) heartbeat(w http.ResponseWriter, r *http.Request) {
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
	response, err := a.store.ReceiveHeartbeat(r.Context(), deviceID, request, a.now())
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, storage.ErrRevisionAhead) {
			status = http.StatusConflict
		}
		writeJSONError(w, status, "heartbeat rejected")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(response)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(protocol.ErrorResponse{Error: message})
}
