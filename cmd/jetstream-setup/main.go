// Command jetstream-setup configures the NATS resources used by Voidcut.
package main

import (
	"log"

	"github.com/devwillsha/voidcut/internal/config"
	"github.com/devwillsha/voidcut/internal/logging"
	messaging "github.com/devwillsha/voidcut/internal/messaging/nats"
	"go.uber.org/zap"
)

func main() {
	cfg := config.Load()
	logger, err := logging.New("jetstream-setup")
	if err != nil {
		log.Fatalf("initialize logger: %v", err)
	}
	defer logger.Sync()

	publisher, err := messaging.Connect(cfg.NATSURL)
	if err != nil {
		logger.Fatal("connect to NATS", zap.Error(err))
	}
	defer publisher.Close()

	resources, err := messaging.ConfigureJetStream(publisher.Connection())
	if err != nil {
		logger.Fatal("configure JetStream", zap.Error(err))
	}
	logger.Info("JetStream resources configured",
		zap.String("stream", resources.StreamName),
		zap.String("consumer", resources.ConsumerName),
		zap.String("dead_letter_stream", resources.DeadLetterStreamName),
	)
}
