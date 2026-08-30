package logging

import (
	"errors"
	"testing"
)

func TestNewRequiresServiceName(t *testing.T) {
	if _, err := New(""); !errors.Is(err, ErrServiceRequired) {
		t.Fatalf("New(\"\") error = %v, want ErrServiceRequired", err)
	}
}

func TestNewCreatesLogger(t *testing.T) {
	logger, err := New("agent")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if logger == nil {
		t.Fatal("New() returned nil logger")
	}
	_ = logger.Sync()
}
