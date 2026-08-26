package schema

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVersionedSubjects(t *testing.T) {
	subjects := []Subject{ActivityEventsV1, JobCreatedV1, JobUpdatedV1, JobDoneV1}
	for _, subject := range subjects {
		if len(subject) < 3 || subject[len(subject)-2:] != "v1" {
			t.Errorf("subject %q is not versioned", subject)
		}
	}
}

func TestEventEnvelopeJSONContract(t *testing.T) {
	event := EventEnvelope{
		EventID:   "event-1",
		TraceID:   "trace-1",
		UserID:    "user-1",
		SessionID: "session-1",
		DeviceID:  "device-1",
		EventType: "keyboard",
		Version:   VersionV1,
		Timestamp: time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC),
		Payload:   json.RawMessage(`{"key":"a"}`),
	}

	raw, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal EventEnvelope: %v", err)
	}

	var decoded EventEnvelope
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal EventEnvelope: %v", err)
	}
	if decoded.EventID != event.EventID || decoded.TraceID != event.TraceID || decoded.UserID != event.UserID || decoded.SessionID != event.SessionID || decoded.DeviceID != event.DeviceID || decoded.Version != VersionV1 {
		t.Fatalf("envelope correlation or version fields changed: %+v", decoded)
	}
}
