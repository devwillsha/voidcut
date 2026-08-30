//go:build portaudio

package microphone

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const audioBufferCapacity = 4

// PortAudioSource adapts the default input device to SampleSource.
type PortAudioSource struct {
	stream *portaudio.Stream
	buffer chan []float32
	once   sync.Once
}

// NewPortAudioSource opens the default microphone and starts a callback stream.
func NewPortAudioSource(sampleRate, chunkSize int) (SampleSource, error) {
	if sampleRate <= 0 || chunkSize <= 0 {
		return nil, errors.New("sample rate and chunk size must be positive")
	}
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("initialize PortAudio: %w", err)
	}

	device, err := portaudio.DefaultInputDevice()
	if err != nil {
		_ = portaudio.Terminate()
		return nil, fmt.Errorf("find default microphone: %w", err)
	}
	params := portaudio.LowLatencyParameters(device, nil)
	params.Input.Channels = 1
	params.SampleRate = float64(sampleRate)
	params.FramesPerBuffer = chunkSize
	buffer := make(chan []float32, audioBufferCapacity)
	source := &PortAudioSource{buffer: buffer}
	stream, err := portaudio.OpenStream(params, func(samples []float32) {
		chunk := append([]float32(nil), samples...)
		select {
		case buffer <- chunk:
		default:
			// Drop the oldest chunk when the consumer falls behind.
			select {
			case <-buffer:
			default:
			}
			select {
			case buffer <- chunk:
			default:
			}
		}
	})
	if err != nil {
		_ = portaudio.Terminate()
		return nil, fmt.Errorf("open microphone stream: %w", err)
	}
	source.stream = stream
	if err := source.stream.Start(); err != nil {
		_ = source.stream.Close()
		_ = portaudio.Terminate()
		return nil, fmt.Errorf("start microphone stream: %w", err)
	}
	return source, nil
}

// Read returns the next callback chunk or stops when ctx is cancelled.
func (source *PortAudioSource) Read(ctx context.Context) ([]float32, error) {
	if source == nil || source.stream == nil {
		return nil, errors.New("PortAudio source is not initialized")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case samples := <-source.buffer:
		return samples, nil
	}
}

// Close stops and releases the PortAudio stream exactly once.
func (source *PortAudioSource) Close() error {
	if source == nil {
		return nil
	}
	var closeErr error
	source.once.Do(func() {
		if err := source.stream.Stop(); err != nil {
			closeErr = fmt.Errorf("stop microphone stream: %w", err)
		}
		if err := source.stream.Close(); err != nil && closeErr == nil {
			closeErr = fmt.Errorf("close microphone stream: %w", err)
		}
		if err := portaudio.Terminate(); err != nil && closeErr == nil {
			closeErr = fmt.Errorf("terminate PortAudio: %w", err)
		}
	})
	return closeErr
}
