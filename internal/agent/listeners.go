package agent

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/devwillsha/voidcut/internal/listeners/input"
	"github.com/devwillsha/voidcut/internal/listeners/microphone"
	"github.com/devwillsha/voidcut/internal/schema"
)

// RunListeners runs microphone and keyboard/mouse listeners concurrently.
// Both listeners share the same publish handler and stop when ctx is cancelled
// or either listener returns an error.
func RunListeners(ctx context.Context, microphoneSource microphone.SampleSource, inputSource input.Source, microphoneConfig microphone.Config, inputConfig input.Config, publish func(schema.EventEnvelope) error) error {
	if ctx == nil {
		return errors.New("context is required")
	}
	if microphoneSource == nil {
		return errors.New("microphone source is required")
	}
	if inputSource == nil {
		return errors.New("input source is required")
	}
	if publish == nil {
		return errors.New("event publisher is required")
	}

	runContext, cancel := context.WithCancel(ctx)
	defer cancel()

	events := make(chan error, 2)
	var listeners sync.WaitGroup
	listeners.Add(2)
	go func() {
		defer listeners.Done()
		listener, err := microphone.New(microphoneSource, microphoneConfig, publish)
		if err != nil {
			events <- err
			return
		}
		events <- listener.Run(runContext)
	}()
	go func() {
		defer listeners.Done()
		listener, err := input.New(inputSource, inputConfig, publish)
		if err != nil {
			events <- err
			return
		}
		events <- listener.Run(runContext)
	}()

	var firstErr error
	completed := 0
	for completed < 2 {
		select {
		case err := <-events:
			completed++
			if firstErr == nil && err != nil && !errors.Is(err, context.Canceled) {
				firstErr = err
				cancel()
			}
		case <-ctx.Done():
			cancel()
			completed = 2
		}
	}

	_ = microphoneSource.Close()
	_ = inputSource.Close()
	listeners.Wait()

	for range 2 - completed {
		select {
		case err := <-events:
			if firstErr == nil && err != nil && !errors.Is(err, context.Canceled) {
				firstErr = err
			}
		default:
		}
	}
	if firstErr != nil && !errors.Is(firstErr, context.Canceled) {
		return fmt.Errorf("listener stopped: %w", firstErr)
	}
	return nil
}
