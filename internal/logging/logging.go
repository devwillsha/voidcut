// Package logging provides a shared structured logger for all binaries.
package logging

import (
	"errors"
	"strings"

	"go.uber.org/zap"
)

var ErrServiceRequired = errors.New("service name is required")

// New builds a zap logger tagged with the given service name, e.g. "agent"
// or "gateway". Every binary should call this once at startup.
func New(service string) (*zap.Logger, error) {
	if strings.TrimSpace(service) == "" {
		return nil, ErrServiceRequired
	}
	logger, err := zap.NewProduction(
		zap.Fields(zap.String("service", service)),
	)
	if err != nil {
		return nil, err
	}
	return logger, nil
}
