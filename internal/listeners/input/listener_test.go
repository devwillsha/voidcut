package input

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/schema"
)

type eventSource struct {
	events []Event
	index  int
}

func (source *eventSource) Read(context.Context) (Event, error) {
	if source.index >= len(source.events) {
		return Event{}, context.Canceled
	}
	event := source.events[source.index]
	source.index++
	return event, nil
}

func (source *eventSource) Close() error { return nil }

func TestListenerEmitsKeyboardAndMouseEvents(t *testing.T) {
	source := &eventSource{events: []Event{
		{Type: Keyboard, Key: "a"},
		{Type: Mouse, Action: "click", X: 100, Y: 200, Meta: map[string]string{"button": "left"}},
	}}
	var received []schema.EventEnvelope
	listener, err := New(source, Config{
		UserID:    "user-1",
		SessionID: "session-1",
		DeviceID:  "device-1",
		TraceID:   "trace-1",
		Timestamp: func() time.Time { return time.Unix(100, 0) },
	}, func(event schema.EventEnvelope) error {
		received = append(received, event)
		return nil
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := listener.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(received) != 2 {
		t.Fatalf("received %d events, want 2", len(received))
	}
	if received[0].EventType != string(events.KeyboardSource) || received[1].EventType != "mouse" {
		t.Fatalf("unexpected event types: %q, %q", received[0].EventType, received[1].EventType)
	}

	var mouse events.MouseActivity
	if err := json.Unmarshal(received[1].Payload, &mouse); err != nil {
		t.Fatalf("unmarshal mouse payload: %v", err)
	}
	if mouse.Action != "click" || mouse.X != 100 || mouse.Y != 200 || mouse.Meta["source"] != "mouse" {
		t.Fatalf("unexpected mouse payload: %+v", mouse)
	}
}

func TestListenerRejectsUnsupportedInput(t *testing.T) {
	listener, err := New(&eventSource{events: []Event{{Type: "touch"}}}, Config{}, func(schema.EventEnvelope) error { return nil })
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := listener.Run(context.Background()); err == nil {
		t.Fatal("Run() returned nil for unsupported input")
	}
}

func TestListenerReturnsHandlerErrors(t *testing.T) {
	handlerErr := errors.New("publish failed")
	listener, err := New(&eventSource{events: []Event{{Type: Keyboard, Key: "a"}}}, Config{}, func(schema.EventEnvelope) error { return handlerErr })
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := listener.Run(context.Background()); !errors.Is(err, handlerErr) {
		t.Fatalf("Run() error = %v, want %v", err, handlerErr)
	}
}

func TestNewRejectsInvalidDependencies(t *testing.T) {
	if _, err := New(nil, Config{}, func(schema.EventEnvelope) error { return nil }); err == nil {
		t.Fatal("New(nil, ...) returned nil error")
	}
	if _, err := New(&eventSource{}, Config{}, nil); err == nil {
		t.Fatal("New(..., nil) returned nil error")
	}
}
