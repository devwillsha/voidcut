package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/devwillsha/voidcut/internal/listeners/input"
	"github.com/devwillsha/voidcut/internal/listeners/microphone"
	"github.com/devwillsha/voidcut/internal/schema"
)

type microphoneSource struct {
	closed bool
}

func (source *microphoneSource) Read(ctx context.Context) ([]float32, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}

func (source *microphoneSource) Close() error {
	source.closed = true
	return nil
}

type inputSource struct {
	closed bool
}

func (source *inputSource) Read(ctx context.Context) (input.Event, error) {
	<-ctx.Done()
	return input.Event{}, ctx.Err()
}

func (source *inputSource) Close() error {
	source.closed = true
	return nil
}

func TestRunListenersStopsAndClosesSources(t *testing.T) {
	microphoneSource := &microphoneSource{}
	inputSource := &inputSource{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := RunListeners(ctx, microphoneSource, inputSource,
		microphone.Config{SampleRate: 48000, ChunkSize: 480},
		input.Config{},
		func(schema.EventEnvelope) error { return nil },
	)
	if err != nil {
		t.Fatalf("RunListeners() error = %v", err)
	}
	if !microphoneSource.closed || !inputSource.closed {
		t.Fatal("RunListeners() did not close both sources")
	}
}

func TestRunListenersPropagatesListenerError(t *testing.T) {
	listenerErr := errors.New("source failed")
	microphoneSource := &errorMicrophoneSource{err: listenerErr}
	inputSource := &inputSource{}

	err := RunListeners(context.Background(), microphoneSource, inputSource,
		microphone.Config{SampleRate: 48000, ChunkSize: 480},
		input.Config{},
		func(schema.EventEnvelope) error { return nil },
	)
	if !errors.Is(err, listenerErr) {
		t.Fatalf("RunListeners() error = %v, want %v", err, listenerErr)
	}
}

type errorMicrophoneSource struct {
	err error
}

func (source *errorMicrophoneSource) Read(context.Context) ([]float32, error) {
	return nil, source.err
}

func (source *errorMicrophoneSource) Close() error { return nil }

func TestNewID(t *testing.T) {
	first, err := NewID("session")
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewID("session")
	if err != nil {
		t.Fatal(err)
	}
	if first == second || len(first) <= len("session-") {
		t.Fatalf("generated IDs = %q and %q", first, second)
	}
}
