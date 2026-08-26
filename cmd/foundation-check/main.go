// Command foundation-check validates live keyboard, mouse, and microphone events.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
	"github.com/devwillsha/voidcut/internal/verification"
	natsgo "github.com/nats-io/nats.go"
)

func main() {
	natsURL := flag.String("nats-url", "nats://localhost:4222", "NATS server URL")
	timeout := flag.Duration("timeout", 60*time.Second, "maximum time to wait for all event types")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	connection, err := natsgo.Connect(*natsURL)
	if err != nil {
		fail("connect to NATS", err)
	}
	defer connection.Close()

	subscription, err := connection.SubscribeSync(string(schema.ActivityEventsV1))
	if err != nil {
		fail("subscribe to activity events", err)
	}
	defer subscription.Unsubscribe()
	fmt.Printf("listening on %s for microphone, keyboard, and mouse events\n", schema.ActivityEventsV1)

	seen := make(map[string]bool, len(verification.RequiredEventTypes))
	for len(seen) < len(verification.RequiredEventTypes) {
		message, err := nextMessage(ctx, subscription)
		if err != nil {
			fail("wait for activity event", err)
		}
		event, err := verification.ValidateActivityEvent(message.Data)
		if err != nil {
			fmt.Printf("ignored invalid event: %v\n", err)
			continue
		}
		if !seen[event.EventType] {
			seen[event.EventType] = true
			fmt.Printf("received %-10s event: user=%s session=%s trace=%s\n", event.EventType, event.UserID, event.SessionID, event.TraceID)
		}
	}
	fmt.Println("foundation verification passed")
}

func nextMessage(ctx context.Context, subscription *natsgo.Subscription) (*natsgo.Msg, error) {
	for {
		message, err := subscription.NextMsg(250 * time.Millisecond)
		if err == nil {
			return message, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
	}
}

func fail(operation string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", operation, err)
	os.Exit(1)
}
