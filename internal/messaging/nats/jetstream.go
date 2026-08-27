package nats

import (
	"errors"
	"fmt"
	"time"

	"github.com/devwillsha/voidcut/internal/schema"
	natsgo "github.com/nats-io/nats.go"
)

const (
	activityStreamName        = "ACTIVITY_EVENTS"
	activityDeadLetterStream  = "ACTIVITY_EVENTS_DLQ"
	activityConsumerName      = "ACTIVITY_PERSISTOR"
	activityDeadLetterSubject = "activity.events.v1.dlq"
)

// JetStreamResources identifies the activity stream and consumer used by the
// services that persist activity events.
type JetStreamResources struct {
	StreamName           string
	ConsumerName         string
	DeadLetterStreamName string
}

// ConfigureJetStream creates the durable activity stream, its dead-letter
// stream, and the consumer used by the Activity service. Re-running it keeps
// the existing resources in place.
func ConfigureJetStream(connection *natsgo.Conn) (JetStreamResources, error) {
	if connection == nil {
		return JetStreamResources{}, errors.New("NATS connection is required")
	}

	jetStream, err := connection.JetStream()
	if err != nil {
		return JetStreamResources{}, fmt.Errorf("create JetStream context: %w", err)
	}

	if err := ensureStream(jetStream, &natsgo.StreamConfig{
		Name:      activityStreamName,
		Subjects:  []string{string(schema.ActivityEventsV1)},
		Retention: natsgo.LimitsPolicy,
		Storage:   natsgo.FileStorage,
		MaxAge:    7 * 24 * time.Hour,
	}); err != nil {
		return JetStreamResources{}, fmt.Errorf("configure activity stream: %w", err)
	}

	if err := ensureStream(jetStream, &natsgo.StreamConfig{
		Name:      activityDeadLetterStream,
		Subjects:  []string{activityDeadLetterSubject},
		Retention: natsgo.LimitsPolicy,
		Storage:   natsgo.FileStorage,
		MaxAge:    30 * 24 * time.Hour,
	}); err != nil {
		return JetStreamResources{}, fmt.Errorf("configure activity dead-letter stream: %w", err)
	}

	if err := ensureConsumer(jetStream, activityStreamName, &natsgo.ConsumerConfig{
		Durable:       activityConsumerName,
		AckPolicy:     natsgo.AckExplicitPolicy,
		AckWait:       30 * time.Second,
		MaxDeliver:    5,
		DeliverPolicy: natsgo.DeliverAllPolicy,
		ReplayPolicy:  natsgo.ReplayInstantPolicy,
	}); err != nil {
		return JetStreamResources{}, fmt.Errorf("configure activity consumer: %w", err)
	}

	return JetStreamResources{
		StreamName:           activityStreamName,
		ConsumerName:         activityConsumerName,
		DeadLetterStreamName: activityDeadLetterStream,
	}, nil
}

func ensureStream(jetStream natsgo.JetStreamContext, config *natsgo.StreamConfig) error {
	_, err := jetStream.AddStream(config)
	if err == nil || errors.Is(err, natsgo.ErrStreamNameAlreadyInUse) {
		return nil
	}
	return err
}

func ensureConsumer(jetStream natsgo.JetStreamContext, streamName string, config *natsgo.ConsumerConfig) error {
	_, err := jetStream.AddConsumer(streamName, config)
	if err == nil || errors.Is(err, natsgo.ErrConsumerNameAlreadyInUse) {
		return nil
	}
	return err
}
