// Package nats publishes Voidcut event envelopes to NATS.
package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
	natsgo "github.com/nats-io/nats.go"
)

// Publisher publishes versioned event envelopes over NATS.
type Publisher struct {
	connection *natsgo.Conn
}

// Connect opens a NATS connection for activity and job event publishing.
func Connect(url string) (*Publisher, error) {
	if strings.TrimSpace(url) == "" {
		return nil, errors.New("NATS URL is required")
	}
	connection, err := natsgo.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}
	return &Publisher{connection: connection}, nil
}

// PublishActivity publishes one event to the versioned activity subject.
func (publisher *Publisher) PublishActivity(ctx context.Context, event schema.EventEnvelope) error {
	if publisher == nil || publisher.connection == nil {
		return errors.New("NATS publisher is not connected")
	}
	if err := validateContext(ctx); err != nil {
		return err
	}
	if event.EventID == "" || event.TraceID == "" || event.UserID == "" || event.SessionID == "" || event.DeviceID == "" || event.EventType == "" || event.Version == "" || event.Timestamp.IsZero() || !json.Valid(event.Payload) {
		return errors.New("event envelope is incomplete or has invalid JSON payload")
	}
	raw, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode event envelope: %w", err)
	}
	if err := publisher.connection.Publish(string(schema.ActivityEventsV1), raw); err != nil {
		return fmt.Errorf("publish activity event: %w", err)
	}
	return nil
}

// Flush waits for all buffered publications to reach the NATS server.
func (publisher *Publisher) Flush(ctx context.Context) error {
	if publisher == nil || publisher.connection == nil {
		return errors.New("NATS publisher is not connected")
	}
	if err := validateContext(ctx); err != nil {
		return err
	}
	if err := publisher.connection.FlushTimeout(time.Until(deadlineFromContext(ctx))); err != nil {
		return fmt.Errorf("flush NATS publications: %w", err)
	}
	return nil
}

func deadlineFromContext(ctx context.Context) time.Time {
	deadline, ok := ctx.Deadline()
	if ok {
		return deadline
	}
	return time.Now().Add(10 * time.Second)
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// Close flushes pending publications and closes the NATS connection.
func (publisher *Publisher) Close() error {
	if publisher == nil || publisher.connection == nil {
		return nil
	}
	publisher.connection.Close()
	return nil
}
