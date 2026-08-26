// Package verification contains checks for the live foundation workflow.
package verification

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/devwillsha/voidcut/internal/schema"
)

// RequiredEventTypes are the activity types needed for a complete hardware check.
var RequiredEventTypes = map[string]struct{}{
	"microphone": {},
	"keyboard":   {},
	"mouse":      {},
}

// ValidateActivityEvent checks the envelope fields required by the live test.
func ValidateActivityEvent(data []byte) (schema.EventEnvelope, error) {
	var event schema.EventEnvelope
	if err := json.Unmarshal(data, &event); err != nil {
		return schema.EventEnvelope{}, fmt.Errorf("decode activity event: %w", err)
	}
	if event.EventID == "" || event.TraceID == "" || event.UserID == "" || event.SessionID == "" || event.DeviceID == "" || event.EventType == "" || event.Version == "" || event.Timestamp.IsZero() {
		return schema.EventEnvelope{}, errors.New("activity event is missing required envelope fields")
	}
	if _, ok := RequiredEventTypes[event.EventType]; !ok {
		return schema.EventEnvelope{}, fmt.Errorf("unexpected activity event type %q", event.EventType)
	}
	if !json.Valid(event.Payload) {
		return schema.EventEnvelope{}, errors.New("activity event payload is invalid JSON")
	}
	return event, nil
}
