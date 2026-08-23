// Command agent is the local recording agent: mic/keyboard listeners,
// device-login, and activity event publishing. Logic is added in later tasks.
package main

import (
	"log"

	"github.com/devwillsha/voidcut/internal/config"
	"github.com/devwillsha/voidcut/internal/logging"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	logger, err := logging.New("agent")
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("agent starting",
		zap.String("env", cfg.Env),
		zap.String("nats_url", cfg.NATSURL),
	)
}
