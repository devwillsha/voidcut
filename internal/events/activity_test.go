package events

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
)

func TestNewKeyboardEvent(t *testing.T) {
	ts := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	event, err := NewKeyboardEvent("user-1", "session-1", "device-1", "trace-1", "a", ts)
	if err != nil {
		t.Fatalf("NewKeyboardEvent() error = %v", err)
	}

	if event.EventID != "evt-trace-1-keyboard" || event.EventType != string(KeyboardSource) {
		t.Fatalf("unexpected event identity: %+v", event)
	}
	if event.TraceID != "trace-1" || event.UserID != "user-1" || event.SessionID != "session-1" || event.DeviceID != "device-1" {
		t.Fatalf("correlation fields were not preserved: %+v", event)
	}
	if event.Version != schema.VersionV1 || !event.Timestamp.Equal(ts) {
		t.Fatalf("unexpected version or timestamp: %+v", event)
	}

	var payload KeyboardActivity
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal keyboard payload: %v", err)
	}
	if payload.Key != "a" || payload.Meta["source"] != string(KeyboardSource) {
		t.Fatalf("unexpected keyboard payload: %+v", payload)
	}
}

func TestNewMicrophoneEvent(t *testing.T) {
	event, err := NewMicrophoneEvent("user-1", "session-1", "device-1", "trace-1", 48000, 250, time.Time{})
	if err != nil {
		t.Fatalf("NewMicrophoneEvent() error = %v", err)
	}

	var payload MicrophoneActivity
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("unmarshal microphone payload: %v", err)
	}
	if payload.SampleRate != 48000 || payload.DurationMS != 250 || payload.Meta["source"] != string(MicSource) {
		t.Fatalf("unexpected microphone payload: %+v", payload)
	}
}
