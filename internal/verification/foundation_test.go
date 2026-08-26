package verification

import (
	"encoding/json"
	"testing"
	"time"
)

func TestValidateActivityEvent(t *testing.T) {
	raw, err := json.Marshal(map[string]interface{}{
		"event_id": "event-1", "trace_id": "trace-1", "user_id": "user-1",
		"session_id": "session-1", "device_id": "device-1", "event_type": "microphone",
		"version": "v1", "timestamp": time.Now().UTC(), "payload": map[string]interface{}{"duration_ms": 10},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateActivityEvent(raw); err != nil {
		t.Fatalf("ValidateActivityEvent() error = %v", err)
	}
}

func TestValidateActivityEventRejectsInvalidInput(t *testing.T) {
	cases := [][]byte{
		[]byte("{"),
		[]byte(`{"event_type":"microphone"}`),
		[]byte(`{"event_id":"event-1","trace_id":"trace-1","user_id":"user-1","session_id":"session-1","device_id":"device-1","event_type":"video","version":"v1","timestamp":"2026-08-26T12:00:00Z","payload":{}}`),
	}
	for _, input := range cases {
		if _, err := ValidateActivityEvent(input); err == nil {
			t.Errorf("ValidateActivityEvent(%s) returned nil", input)
		}
	}
}
