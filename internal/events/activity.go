// Package events contains typed activity payloads that are sent over NATS and
// used by downstream services.
package events

import (
	"encoding/json"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
)

// ActivitySource identifies the sensor or input that produced an event.
type ActivitySource string

const (
	KeyboardSource ActivitySource = "keyboard"
	MicSource      ActivitySource = "microphone"
)

// KeyboardActivity is the payload produced by the keyboard listener.
type KeyboardActivity struct {
	Key  string            `json:"key"`
	Meta map[string]string `json:"meta"`
}

// MicrophoneActivity is the payload produced by the microphone listener.
type MicrophoneActivity struct {
	SampleRate int               `json:"sample_rate"`
	DurationMS int               `json:"duration_ms"`
	Meta       map[string]string `json:"meta"`
}

// NewKeyboardEvent builds a typed keyboard event envelope using the shared
// schema contract.
func NewKeyboardEvent(userID, sessionID, deviceID, traceID, key string, ts time.Time) (schema.EventEnvelope, error) {
	payload := KeyboardActivity{
		Key: key,
		Meta: map[string]string{
			"source": string(KeyboardSource),
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return schema.EventEnvelope{}, err
	}

	return schema.EventEnvelope{
		EventID:   "evt-" + traceID + "-keyboard",
		TraceID:   traceID,
		UserID:    userID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		EventType: string(KeyboardSource),
		Version:   schema.VersionV1,
		Timestamp: ts,
		Payload:   raw,
	}, nil
}

// NewMicrophoneEvent builds a typed microphone event envelope using the shared
// schema contract.
func NewMicrophoneEvent(userID, sessionID, deviceID, traceID string, sampleRate, durationMS int, ts time.Time) (schema.EventEnvelope, error) {
	payload := MicrophoneActivity{
		SampleRate: sampleRate,
		DurationMS: durationMS,
		Meta: map[string]string{
			"source": string(MicSource),
		},
	}

	raw, err := json.Marshal(payload)
	if err != nil {
		return schema.EventEnvelope{}, err
	}

	return schema.EventEnvelope{
		EventID:   "evt-" + traceID + "-mic",
		TraceID:   traceID,
		UserID:    userID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		EventType: string(MicSource),
		Version:   schema.VersionV1,
		Timestamp: ts,
		Payload:   raw,
	}, nil
}
