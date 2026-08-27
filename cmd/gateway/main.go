// Command gateway is the API Gateway: HTTP routing, auth middleware, and
// request logging for all client-facing traffic. Routes are added in later tasks.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devwillsha/voidcut/internal/auth"
	"github.com/devwillsha/voidcut/internal/config"
	"github.com/devwillsha/voidcut/internal/gateway"
	"github.com/devwillsha/voidcut/internal/logging"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()

	logger, err := logging.New("gateway")
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	sugar.Infow("gateway starting",
		"env", cfg.Env,
		"postgres_dsn_set", cfg.PostgresDSN != "",
	)

	// Create and configure the API Gateway.
	gw, err := gateway.New(":8080", sugar)
	if err != nil {
		sugar.Fatalf("failed to create gateway: %v", err)
	}

	// Mount health check endpoints.
	gw.MountReadiness()
	gw.MountLiveness()

	// Initialize Redis and device login service.
	redis := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	deviceSvc, err := auth.NewDeviceLoginService(redis)
	if err != nil {
		sugar.Fatalf("failed to create device login service: %v", err)
	}
	gw.MountDeviceAuthRoutes(deviceSvc)
	defer redis.Close()

	// Start the gateway in a background goroutine.
	errChan := make(chan error, 1)
	go func() {
		errChan <- gw.Start()
	}()

	// Wait for shutdown signal or startup error.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigChan:
		sugar.Infow("received signal, shutting down", "signal", sig)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = ctx
		// The Gateway does not have a built-in graceful shutdown for the listener.
		// In production, we would use http.Server with Shutdown() method (see Phase 3).
		_ = gw.Stop()
	case err := <-errChan:
		if err != nil {
			sugar.Fatalf("gateway error: %v", err)
		}
	}
}
