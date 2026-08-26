//go:build !gohook

package input

import (
	"errors"
	"testing"
)

func TestNewGlobalSourceRequiresGohookBuild(t *testing.T) {
	_, err := NewGlobalSource()
	if !errors.Is(err, ErrOSInputUnavailable) {
		t.Fatalf("NewGlobalSource() error = %v, want ErrOSInputUnavailable", err)
	}
}
