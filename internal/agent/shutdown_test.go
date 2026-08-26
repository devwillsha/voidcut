package agent

import (
	"context"
	"errors"
	"testing"
)

func TestGracefulShutdownFlushesBeforeClosing(t *testing.T) {
	var calls []string
	if err := GracefulShutdown(func(ctx context.Context) error {
		if _, ok := ctx.Deadline(); !ok {
			t.Fatal("flush context has no deadline")
		}
		calls = append(calls, "flush")
		return nil
	}, func() error {
		calls = append(calls, "close")
		return nil
	}); err != nil {
		t.Fatalf("GracefulShutdown() error = %v", err)
	}
	if len(calls) != 2 || calls[0] != "flush" || calls[1] != "close" {
		t.Fatalf("shutdown calls = %v, want [flush close]", calls)
	}
}

func TestGracefulShutdownClosesAfterFlushError(t *testing.T) {
	flushErr := errors.New("flush failed")
	closeErr := errors.New("close failed")
	if err := GracefulShutdown(func(context.Context) error { return flushErr }, func() error { return closeErr }); !errors.Is(err, flushErr) || !errors.Is(err, closeErr) {
		t.Fatalf("GracefulShutdown() error = %v, want both errors", err)
	}
}

func TestGracefulShutdownAllowsNilCallbacks(t *testing.T) {
	if err := GracefulShutdown(nil, nil); err != nil {
		t.Fatalf("GracefulShutdown() error = %v", err)
	}
}
