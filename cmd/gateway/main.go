// Command gateway is the API Gateway: HTTP routing, auth middleware, and
// request logging for all client-facing traffic. Routes are added in later tasks.
package main

import (
	"log"

	"github.com/devwillsha/voidcut/internal/config"
	"github.com/devwillsha/voidcut/internal/logging"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()

	logger, err := logging.New("gateway")
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("gateway starting",
		zap.String("env", cfg.Env),
		zap.Bool("postgres_dsn_set", cfg.PostgresDSN != ""),
	)
}
