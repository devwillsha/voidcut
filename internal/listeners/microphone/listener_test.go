// Package microphone tests the sample processing and listener lifecycle.
package microphone

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/schema"
)

type sampleSource struct {
	chunks [][]float32
	index  int
}

func (source *sampleSource) Read(ctx context.Context) ([]float32, error) {
	if source.index >= len(source.chunks) {
		return nil, context.Canceled
	}
	chunk := source.chunks[source.index]
	source.index++
	return chunk, nil
}

func (source *sampleSource) Close() error { return nil }

func TestRMS(t *testing.T) {
	if got := RMS([]float32{1, -1, 1, -1}); got != 1 {
		t.Fatalf("RMS() = %v, want 1", got)
	}
	if got := RMS(nil); got != 0 {
		t.Fatalf("RMS(nil) = %v, want 0", got)
	}
}

func TestListenerEmitsOnlyAboveThreshold(t *testing.T) {
	source := &sampleSource{chunks: [][]float32{
		{0.1, 0.1, 0.1, 0.1},
		{0.8, -0.8, 0.8, -0.8},
	}}
	var received []schema.EventEnvelope
	listener, err := New(source, Config{
		SampleRate: 1000,
		ChunkSize:  4,
		Threshold:  0.5,
		UserID:     "user-1",
		SessionID:  "session-1",
		DeviceID:   "device-1",
		TraceID:    "trace-1",
		Timestamp:  func() time.Time { return time.Unix(100, 0) },
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
	if len(received) != 1 {
		t.Fatalf("received %d events, want 1", len(received))
	}
	if received[0].UserID != "user-1" || received[0].EventType != string(events.MicSource) {
		t.Fatalf("unexpected event: %+v", received[0])
	}
}

func TestListenerReturnsHandlerErrors(t *testing.T) {
	handlerErr := errors.New("publish failed")
	listener, err := New(&sampleSource{chunks: [][]float32{{1}}}, Config{
		SampleRate: 1,
		ChunkSize:  1,
		Threshold:  0.5,
	}, func(schema.EventEnvelope) error { return handlerErr })
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := listener.Run(context.Background()); !errors.Is(err, handlerErr) {
		t.Fatalf("Run() error = %v, want %v", err, handlerErr)
	}
}

func TestNewRejectsInvalidConfiguration(t *testing.T) {
	valid := Config{SampleRate: 1, ChunkSize: 1}
	for name, config := range map[string]Config{
		"nil source": valid,
		"bad rate":   {ChunkSize: 1},
		"bad chunk":  {SampleRate: 1},
		"bad limit":  {SampleRate: 1, ChunkSize: 1, Threshold: -1},
	} {
		var source SampleSource
		if name != "nil source" {
			source = &sampleSource{}
		}
		if _, err := New(source, config, func(schema.EventEnvelope) error { return nil }); err == nil {
			t.Errorf("New(%s) returned nil error", name)
		}
	}
}
