// Command agent is the local recording agent: mic/keyboard listeners,
// device-login, and activity event publishing. Logic is added in later tasks.
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	agentlifecycle "github.com/devwillsha/voidcut/internal/agent"
	"github.com/devwillsha/voidcut/internal/auth"
	"github.com/devwillsha/voidcut/internal/config"
	"github.com/devwillsha/voidcut/internal/listeners/input"
	"github.com/devwillsha/voidcut/internal/listeners/microphone"
	"github.com/devwillsha/voidcut/internal/logging"
	natspublisher "github.com/devwillsha/voidcut/internal/messaging/nats"
	"github.com/devwillsha/voidcut/internal/schema"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	runContext, stop := agentlifecycle.SignalContext(context.Background())
	defer stop()

	logger, err := logging.New("agent")
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("agent starting",
		zap.Int("pid", os.Getpid()),
		zap.String("env", cfg.Env),
	)

	credentials, err := auth.Load()
	switch {
	case err == nil:
		logger.Info("local credentials loaded",
			zap.String("user_id", credentials.UserID),
		)
	case errors.Is(err, auth.ErrCredentialsNotFound), errors.Is(err, auth.ErrInvalidCredentials):
		logger.Info("local credentials unavailable; requesting device code")
		deviceCode, requestErr := (auth.DeviceLoginClient{BaseURL: cfg.GatewayURL}).Start(runContext)
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
		if openErr := auth.OpenBrowser(runContext, deviceCode.VerificationURL); openErr != nil {
			logger.Warn("could not open verification URL", zap.Error(openErr))
		}
		tokenResponse, pollErr := (auth.DeviceLoginClient{BaseURL: cfg.GatewayURL}).Poll(runContext, deviceCode)
		if pollErr != nil {
			logger.Warn("device login did not complete", zap.Error(pollErr))
			break
		}
		logger.Info("device login approved",
			zap.String("user_id", tokenResponse.UserID),
		)
		credentials, saveErr := auth.CredentialsFromDeviceToken(tokenResponse, time.Now())
		if saveErr != nil {
			logger.Warn("could not prepare local credentials", zap.Error(saveErr))
			break
		}
		if saveErr := auth.Save(credentials); saveErr != nil {
			logger.Warn("could not save local credentials", zap.Error(saveErr))
			break
		}
		logger.Info("local credentials saved",
			zap.String("user_id", credentials.UserID),
		)
	default:
		logger.Warn("could not read local credentials; device login required",
			zap.Error(err),
		)
	}

	activityPublisher, publishErr := natspublisher.Connect(cfg.NATSURL)
	if publishErr != nil {
		logger.Warn("could not connect to NATS", zap.Error(publishErr))
		return
	}
	logger.Info("connected to NATS",
		zap.String("activity_subject", "activity.events.v1"),
	)
	sessionID, idErr := agentlifecycle.NewID("session")
	if idErr != nil {
		logger.Warn("could not create session ID", zap.Error(idErr))
		return
	}
	traceID, idErr := agentlifecycle.NewID("trace")
	if idErr != nil {
		logger.Warn("could not create trace ID", zap.Error(idErr))
		return
	}
	microphoneSource, sourceErr := microphone.NewPortAudioSource(cfg.MicSampleRate, cfg.MicChunkSize)
	if sourceErr != nil {
		logger.Warn("could not start microphone listener", zap.Error(sourceErr))
		return
	}
	inputSource, sourceErr := input.NewGlobalSource()
	if sourceErr != nil {
		_ = microphoneSource.Close()
		logger.Warn("could not start keyboard and mouse listener", zap.Error(sourceErr))
		return
	}
	logger.Info("hardware listeners started",
		zap.String("session_id", sessionID),
		zap.String("trace_id", traceID),
	)
	listenerErr := agentlifecycle.RunListeners(runContext, microphoneSource, inputSource,
		microphone.Config{
			SampleRate: cfg.MicSampleRate,
			ChunkSize:  cfg.MicChunkSize,
			Threshold:  cfg.MicThreshold,
			UserID:     credentials.UserID,
			SessionID:  sessionID,
			DeviceID:   "local-agent",
			TraceID:    traceID,
		},
		input.Config{
			UserID:    credentials.UserID,
			SessionID: sessionID,
			DeviceID:  "local-agent",
			TraceID:   traceID,
		},
		func(event schema.EventEnvelope) error {
			return activityPublisher.PublishActivity(context.Background(), event)
		},
	)
	if listenerErr != nil {
		logger.Warn("hardware listener stopped", zap.Error(listenerErr))
	}
	logger.Info("agent shutdown requested")
	if shutdownErr := agentlifecycle.GracefulShutdown(activityPublisher.Flush, activityPublisher.Close); shutdownErr != nil {
		logger.Warn("agent shutdown completed with errors", zap.Error(shutdownErr))
	}
}
