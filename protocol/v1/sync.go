// Package v1 defines the versioned heartbeat contract shared by server and agent.
package v1

import (
	"encoding/json"
	"time"
)

const HeartbeatPath = "/api/v1/device/heartbeat"

type HeartbeatRequest struct {
	PolicyRevision int64          `json:"policy_revision"`
	LocalDate      string         `json:"local_date"`
	SecondsUsed    int64          `json:"seconds_used"`
	Events         []PendingEvent `json:"events,omitempty"`
	CommandAcks    []string       `json:"command_acks,omitempty"`
}

type PendingEvent struct {
	UUID      string    `json:"uuid"`
	Kind      string    `json:"kind"`
	Payload   jsonRaw   `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type jsonRaw = json.RawMessage

type HeartbeatResponse struct {
	ServerTime         time.Time `json:"server_time"`
	AcknowledgedEvents []string  `json:"acknowledged_events,omitempty"`
	Policy             *Policy   `json:"policy,omitempty"`
	Commands           []Command `json:"commands,omitempty"`
}

type Policy struct {
	Revision              int64     `json:"revision"`
	MonitoringPaused      bool      `json:"monitoring_paused"`
	ManualBlock           bool      `json:"manual_block"`
	WarningMinutes        int       `json:"warning_minutes"`
	LocalPasswordVerifier string    `json:"local_password_verifier,omitempty"`
	WeeklyQuotaSeconds    [7]int64  `json:"weekly_quota_seconds"`
	Routines              []Routine `json:"routines,omitempty"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type Routine struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Days        [7]bool `json:"days"`
	StartSecond int64   `json:"start_second"`
	EndSecond   int64   `json:"end_second"`
	Enabled     bool    `json:"enabled"`
}

type Command struct {
	ID        string    `json:"id"`
	Kind      string    `json:"kind"`
	Payload   jsonRaw   `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
}

type BonusPayload struct {
	UUID      string    `json:"uuid,omitempty"`
	LocalDate string    `json:"local_date"`
	Seconds   int64     `json:"seconds"`
	Origin    string    `json:"origin"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
