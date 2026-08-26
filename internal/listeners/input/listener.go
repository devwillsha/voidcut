// Package input consumes keyboard and mouse activity from an OS or robotgo adapter.
package input

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/schema"
)

// EventType identifies the kind of input event received from the adapter.
type EventType string

const (
	Keyboard EventType = "keyboard"
	Mouse    EventType = "mouse"
)

// Event is a normalized keyboard or mouse event from the input adapter.
type Event struct {
	Type   EventType
	Key    string
	Action string
	X      int
	Y      int
	Meta   map[string]string
}

// Source supplies global input events. An OS syscall or robotgo adapter can
// implement this interface without coupling the listener to one platform.
type Source interface {
	Read(context.Context) (Event, error)
	Close() error
}

// Config contains the correlation fields attached to every input envelope.
type Config struct {
	UserID    string
	SessionID string
	DeviceID  string
	TraceID   string
	Timestamp func() time.Time
}

// EventHandler receives normalized input envelopes.
type EventHandler func(schema.EventEnvelope) error

// Listener consumes input events until cancellation, source exhaustion, or a
// handler error.
type Listener struct {
	source  Source
	config  Config
	handler EventHandler
}

// New creates an input listener with validated configuration.
func New(source Source, config Config, handler EventHandler) (*Listener, error) {
	if source == nil {
		return nil, errors.New("input listener requires a source")
	}
	if handler == nil {
		return nil, errors.New("input listener requires an event handler")
	}
	if config.Timestamp == nil {
		config.Timestamp = time.Now
	}
	return &Listener{source: source, config: config, handler: handler}, nil
}

// Run reads and emits keyboard and mouse events until the context ends.
func (listener *Listener) Run(ctx context.Context) error {
	for {
		inputEvent, err := listener.source.Read(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("read input event: %w", err)
		}

		var event schema.EventEnvelope
		switch inputEvent.Type {
		case Keyboard:
			event, err = events.NewKeyboardEvent(
				listener.config.UserID,
				listener.config.SessionID,
				listener.config.DeviceID,
				listener.config.TraceID,
				inputEvent.Key,
				listener.config.Timestamp(),
			)
		case Mouse:
			event, err = events.NewMouseEvent(
				listener.config.UserID,
				listener.config.SessionID,
				listener.config.DeviceID,
				listener.config.TraceID,
				inputEvent.Action,
				inputEvent.X,
				inputEvent.Y,
				inputEvent.Meta,
				listener.config.Timestamp(),
			)
		default:
			return fmt.Errorf("unsupported input event type %q", inputEvent.Type)
		}
		if err != nil {
			return fmt.Errorf("build input event: %w", err)
		}
		if err := listener.handler(event); err != nil {
			return fmt.Errorf("handle input event: %w", err)
		}
	}
}
