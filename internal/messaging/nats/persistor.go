package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
	natsgo "github.com/nats-io/nats.go"
)

// ActivityInserter is the minimal interface required by the persistor to save
// activity events. This keeps the messaging package decoupled from any
// particular repository implementation and makes testing straightforward.
type ActivityInserter interface {
	Insert(ctx context.Context, event schema.EventEnvelope) error
}

// Persistor reads activity events from the durable JetStream consumer and
// persists them using an ActivityInserter. It handles ACK/NACK and moves
// irrecoverable messages to the configured dead-letter subject.
type Persistor struct {
	js   natsgo.JetStreamContext
	sub  *natsgo.Subscription
	repo ActivityInserter
}

// NewPersistor creates a new Persistor bound to the durable consumer
// configured by ConfigureJetStream. The caller must ensure ConfigureJetStream
// has been run at least once before starting the persistor.
func NewPersistor(conn *natsgo.Conn, repo ActivityInserter) (*Persistor, error) {
	if conn == nil {
		return nil, errors.New("NATS connection is required")
	}
	if repo == nil {
		return nil, errors.New("activity repository is required")
	}
	js, err := conn.JetStream()
	if err != nil {
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	// Bind to the pre-created durable consumer. Use pull subscription so we
	// control fetch intervals and backpressure.
	sub, err := js.PullSubscribe(string(schema.ActivityEventsV1), activityConsumerName, natsgo.BindStream(activityStreamName))
	if err != nil {
		return nil, fmt.Errorf("create pull subscription: %w", err)
	}

	return &Persistor{js: js, sub: sub, repo: repo}, nil
}

// Start runs the fetch loop until the context is cancelled. It returns the
// context error when stopped normally.
func (p *Persistor) Start(ctx context.Context) error {
	if p == nil || p.sub == nil || p.repo == nil {
		return errors.New("persistor is not properly initialized")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		msgs, err := p.sub.Fetch(1, natsgo.MaxWait(1*time.Second))
		if err != nil {
			// Timeout is expected behavior; continue looping.
			if errors.Is(err, natsgo.ErrTimeout) {
				continue
			}
			return fmt.Errorf("fetch messages: %w", err)
		}

		for _, m := range msgs {
			var event schema.EventEnvelope
			if err := json.Unmarshal(m.Data, &event); err != nil {
				// Malformed payloads are terminal; publish to DLQ and ack to
				// prevent redelivery.
				_, _ = p.js.Publish(activityDeadLetterSubject, m.Data)
				_ = m.Ack()
				continue
			}

			if err := p.repo.Insert(ctx, event); err != nil {
				// On repeated delivery failures, publish to DLQ and ack.
				if meta, _ := m.Metadata(); meta != nil && meta.NumDelivered >= 5 {
					_, _ = p.js.Publish(activityDeadLetterSubject, m.Data)
					_ = m.Ack()
					continue
				}
				// Let JetStream retry delivery.
				_ = m.Nak()
				continue
			}

			_ = m.Ack()
		}
	}
}

// Close unsubscribes from the consumer. It does not close the underlying
// NATS connection.
func (p *Persistor) Close() error {
	if p == nil || p.sub == nil {
		return nil
	}
	return p.sub.Unsubscribe()
}
