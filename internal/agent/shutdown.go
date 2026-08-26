// Package agent contains lifecycle helpers for the local agent process.
package agent

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const shutdownTimeout = 10 * time.Second

// SignalContext returns a context cancelled by SIGINT or SIGTERM and a stop
// function that releases the signal notification resources.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// GracefulShutdown flushes pending work before closing the publisher. A fresh
// timeout context is used because the run context is already cancelled by the
// shutdown signal.
func GracefulShutdown(flush func(context.Context) error, closePublisher func() error) error {
	shutdownContext, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	var shutdownErrors []error
	if flush != nil {
		if err := flush(shutdownContext); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}
	if closePublisher != nil {
		if err := closePublisher(); err != nil {
			shutdownErrors = append(shutdownErrors, err)
		}
	}
	return errors.Join(shutdownErrors...)
}
