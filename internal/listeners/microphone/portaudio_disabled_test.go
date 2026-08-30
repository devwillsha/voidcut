//go:build !portaudio

package microphone

import (
	"errors"
	"testing"
)

func TestNewPortAudioSourceRequiresPortAudioBuild(t *testing.T) {
	_, err := NewPortAudioSource(48000, 480)
	if !errors.Is(err, ErrPortAudioUnavailable) {
		t.Fatalf("NewPortAudioSource() error = %v, want ErrPortAudioUnavailable", err)
	}
}
