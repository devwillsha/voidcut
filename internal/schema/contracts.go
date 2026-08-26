// Package schema defines the versioned messaging contract shared by the
// gateway, agent, and background services.
package schema

import (
	"encoding/json"
	"time"
)

// EventVersion identifies the schema version for a message envelope.
type EventVersion string

const (
	// VersionV1 is the initial version for all event envelopes.
	VersionV1 EventVersion = "v1"
)

// EventEnvelope is the base contract shared across NATS messages and internal
// processing. Every service should treat these fields as required.
type EventEnvelope struct {
	EventID   string          `json:"event_id"`
	TraceID   string          `json:"trace_id"`
	UserID    string          `json:"user_id"`
	SessionID string          `json:"session_id"`
	DeviceID  string          `json:"device_id"`
	EventType string          `json:"event_type"`
	Version   EventVersion    `json:"version"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// NATS subject names are explicit and versioned to avoid drift between services.
type Subject string

const (
	ActivityEventsV1 Subject = "activity.events.v1"
	JobCreatedV1     Subject = "job.created.v1"
	JobUpdatedV1     Subject = "job.updated.v1"
	JobDoneV1        Subject = "job.done.v1"
)
