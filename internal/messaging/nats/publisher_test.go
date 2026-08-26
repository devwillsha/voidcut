package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
	natsgo "github.com/nats-io/nats.go"
)

func TestPublisherRejectsIncompleteEvent(t *testing.T) {
	publisher := &Publisher{}
	if err := publisher.PublishActivity(context.Background(), schema.EventEnvelope{}); err == nil {
		t.Fatal("PublishActivity() returned nil error for disconnected publisher")
	}
}

func TestPublisherPublishesActivityEvent(t *testing.T) {
	connection, err := natsgo.Connect("nats://localhost:4222", natsgo.Timeout(500*time.Millisecond))
	if err != nil {
		t.Skipf("local NATS is unavailable: %v", err)
	}
	defer connection.Close()

	publisher := &Publisher{connection: connection}
	subscription, err := connection.SubscribeSync(string(schema.ActivityEventsV1))
	if err != nil {
		t.Fatal(err)
	}
	defer subscription.Unsubscribe()

	event := schema.EventEnvelope{
		EventID:   "event-test-1",
		TraceID:   "trace-test-1",
		UserID:    "user-test-1",
		SessionID: "session-test-1",
		DeviceID:  "device-test-1",
		EventType: "keyboard",
		Version:   schema.VersionV1,
		Timestamp: time.Now().UTC(),
		Payload:   json.RawMessage(`{"key":"a"}`),
	}
	if err := publisher.PublishActivity(context.Background(), event); err != nil {
		t.Fatalf("PublishActivity() error = %v", err)
	}
	if err := connection.Flush(); err != nil {
		t.Fatal(err)
	}

	message, err := subscription.NextMsg(time.Second)
	if err != nil {
		t.Fatalf("NextMsg() error = %v", err)
	}
	var received schema.EventEnvelope
	if err := json.Unmarshal(message.Data, &received); err != nil {
		t.Fatal(err)
	}
	if received.EventID != event.EventID || received.SessionID != event.SessionID {
		t.Fatalf("received event = %+v", received)
	}
}
