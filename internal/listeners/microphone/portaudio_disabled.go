//go:build !portaudio

package microphone

import "errors"

var ErrPortAudioUnavailable = errors.New("PortAudio support requires the portaudio build tag and native PortAudio library")

// NewPortAudioSource is unavailable in the default build. Build with
// -tags portaudio after installing the native PortAudio library.
func NewPortAudioSource(sampleRate, chunkSize int) (SampleSource, error) {
	return nil, ErrPortAudioUnavailable
}
