package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
	natsgo "github.com/nats-io/nats.go"
)

type fakeRepo struct {
	ch chan schema.EventEnvelope
}

func (f *fakeRepo) Insert(ctx context.Context, event schema.EventEnvelope) error {
	select {
	case f.ch <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestPersistorInsertsAndAcks(t *testing.T) {
	conn, err := natsgo.Connect("nats://localhost:4222", natsgo.Timeout(500*time.Millisecond))
	if err != nil {
		t.Skipf("local NATS is unavailable: %v", err)
	}
	defer conn.Close()

	// Ensure JetStream resources exist.
	if _, err := ConfigureJetStream(conn); err != nil {
		t.Fatalf("ConfigureJetStream() error = %v", err)
	}

	js, err := conn.JetStream()
	if err != nil {
		t.Fatal(err)
	}

	// Prepare a fake repository to observe Insert calls.
	ch := make(chan schema.EventEnvelope, 1)
	repo := &fakeRepo{ch: ch}

	persistor, err := NewPersistor(conn, repo)
	if err != nil {
		t.Fatalf("NewPersistor() error = %v", err)
	}
	defer persistor.Close()

	// Publish a valid activity event.
	ev := schema.EventEnvelope{
		EventID:   "evt-1",
		TraceID:   "tr-1",
		UserID:    "user-1",
		SessionID: "sess-1",
		DeviceID:  "dev-1",
		EventType: "mic.sample",
		Version:   schema.VersionV1,
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage("{\"value\":123}"),
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := js.Publish(string(schema.ActivityEventsV1), raw); err != nil {
		t.Fatalf("publish event: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	go func() {
		_ = persistor.Start(ctx)
	}()

	for {
		select {
		case got := <-ch:
			if got.EventID == ev.EventID {
				return
			}
			// ignore unrelated events and continue waiting until timeout
		case <-ctx.Done():
			t.Fatal("persistor did not insert event within timeout")
		}
	}
}
