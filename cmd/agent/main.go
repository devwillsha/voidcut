// Command agent is the local recording agent: mic/keyboard listeners,
// device-login, and activity event publishing. Logic is added in later tasks.
package main

import (
	"context"
	"errors"
	"log"

	"github.com/devwillsha/voidcut/internal/auth"
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

	credentials, err := auth.Load()
	switch {
	case err == nil:
		logger.Info("local credentials loaded",
			zap.String("user_id", credentials.UserID),
		)
	case errors.Is(err, auth.ErrCredentialsNotFound), errors.Is(err, auth.ErrInvalidCredentials):
		logger.Info("local credentials unavailable; requesting device code")
		deviceCode, requestErr := (auth.DeviceLoginClient{BaseURL: cfg.GatewayURL}).Start(context.Background())
		if requestErr != nil {
			logger.Warn("could not request device code", zap.Error(requestErr))
			break
		}
		logger.Info("device code received",
			zap.String("user_code", deviceCode.UserCode),
			zap.String("verification_url", deviceCode.VerificationURL),
			zap.Int("expires_in", deviceCode.ExpiresIn),
			zap.Int("interval", deviceCode.Interval),
		)
		if openErr := auth.OpenBrowser(context.Background(), deviceCode.VerificationURL); openErr != nil {
			logger.Warn("could not open verification URL", zap.Error(openErr))
		}
	default:
		logger.Warn("could not read local credentials; device login required",
			zap.Error(err),
		)
	}
}
