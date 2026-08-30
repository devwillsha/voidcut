package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/auth"
	"github.com/devwillsha/voidcut/internal/events"
	natspublisher "github.com/devwillsha/voidcut/internal/messaging/nats"
	"github.com/devwillsha/voidcut/internal/schema"
	natsgo "github.com/nats-io/nats.go"
)

func TestAgentLoginAndEventStreaming(t *testing.T) {
	gateway := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/auth/device/start":
			_, _ = writer.Write([]byte(`{"device_code":"device-test","user_code":"TEST-1234","verification_url":"https://example.test/connect","expires_in":5,"interval":1}`))
		case "/api/v1/auth/device/token":
			_, _ = writer.Write([]byte(`{"status":"approved","token":"token-test","user_id":"user-test","expires_in":3600}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer gateway.Close()

	client := auth.DeviceLoginClient{BaseURL: gateway.URL}
	deviceCode, err := client.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	token, err := client.Poll(context.Background(), deviceCode)
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if token.UserID != "user-test" || token.Token != "token-test" {
		t.Fatalf("unexpected token response: %+v", token)
	}

	connection, err := natsgo.Connect("nats://localhost:4222", natsgo.Timeout(500*time.Millisecond))
	if err != nil {
		t.Skipf("local NATS is unavailable: %v", err)
	}
	defer connection.Close()

	subscription, err := connection.SubscribeSync(string(schema.ActivityEventsV1))
	if err != nil {
		t.Fatal(err)
	}
	defer subscription.Unsubscribe()

	eventPublisher, err := natspublisher.Connect("nats://localhost:4222")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer eventPublisher.Close()
	event, err := events.NewKeyboardEvent("user-test", "session-test", "device-test", "trace-test", "a", time.Now().UTC())
	if err != nil {
		t.Fatalf("NewKeyboardEvent() error = %v", err)
	}
	if err := eventPublisher.PublishActivity(context.Background(), event); err != nil {
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
		t.Fatalf("decode streamed event: %v", err)
	}
	if received.UserID != "user-test" || received.SessionID != "session-test" || received.TraceID != "trace-test" {
		t.Fatalf("correlation fields were not preserved: %+v", received)
	}
}
