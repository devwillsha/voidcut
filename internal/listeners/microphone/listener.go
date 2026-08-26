// Package microphone detects speech activity from microphone sample chunks.
package microphone

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/schema"
)

// SampleSource supplies successive microphone sample chunks. Implementations
// can wrap PortAudio or a deterministic source used by tests.
type SampleSource interface {
	Read(context.Context) ([]float32, error)
	Close() error
}

// Config controls microphone sampling and speech detection.
type Config struct {
	SampleRate int
	ChunkSize  int
	Threshold  float64
	UserID     string
	SessionID  string
	DeviceID   string
	TraceID    string
	Timestamp  func() time.Time
}

// EventHandler receives microphone events when a chunk exceeds the RMS
// threshold.
type EventHandler func(schema.EventEnvelope) error

// Listener reads microphone chunks until its context is cancelled.
type Listener struct {
	source  SampleSource
	config  Config
	handler EventHandler
}

// New creates a microphone listener with validated configuration.
func New(source SampleSource, config Config, handler EventHandler) (*Listener, error) {
	if source == nil {
		return nil, errors.New("microphone listener requires a sample source")
	}
	if handler == nil {
		return nil, errors.New("microphone listener requires an event handler")
	}
	if config.SampleRate <= 0 {
		return nil, errors.New("sample rate must be positive")
	}
	if config.ChunkSize <= 0 {
		return nil, errors.New("chunk size must be positive")
	}
	if config.Threshold < 0 {
		return nil, errors.New("threshold cannot be negative")
	}
	if config.Timestamp == nil {
		config.Timestamp = time.Now
	}
	return &Listener{source: source, config: config, handler: handler}, nil
}

// Run consumes microphone chunks until cancellation, source exhaustion, or a
// handler error. It returns nil for normal context cancellation.
func (listener *Listener) Run(ctx context.Context) error {
	for {
		samples, err := listener.source.Read(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return fmt.Errorf("read microphone samples: %w", err)
		}
		if len(samples) == 0 {
			continue
		}
		if RMS(samples) < listener.config.Threshold {
			continue
		}

		durationMS := len(samples) * 1000 / listener.config.SampleRate
		if durationMS == 0 {
			durationMS = 1
		}
		event, err := events.NewMicrophoneEvent(
			listener.config.UserID,
			listener.config.SessionID,
			listener.config.DeviceID,
			listener.config.TraceID,
			listener.config.SampleRate,
			durationMS,
			listener.config.Timestamp(),
		)
		if err != nil {
			return fmt.Errorf("build microphone event: %w", err)
		}
		if err := listener.handler(event); err != nil {
			return fmt.Errorf("handle microphone event: %w", err)
		}
	}
}

// RMS returns the root-mean-square amplitude of samples.
func RMS(samples []float32) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, sample := range samples {
		value := float64(sample)
		sum += value * value
	}
	return math.Sqrt(sum / float64(len(samples)))
}
