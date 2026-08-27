package nats

import (
	"testing"
	"time"

	natsgo "github.com/nats-io/nats.go"
)

func TestConfigureJetStream(t *testing.T) {
	connection, err := natsgo.Connect("nats://localhost:4222", natsgo.Timeout(500*time.Millisecond))
	if err != nil {
		t.Skipf("local NATS is unavailable: %v", err)
	}
	defer connection.Close()

	resources, err := ConfigureJetStream(connection)
	if err != nil {
		t.Fatalf("ConfigureJetStream() error = %v", err)
	}
	if resources.StreamName != activityStreamName || resources.ConsumerName != activityConsumerName || resources.DeadLetterStreamName != activityDeadLetterStream {
		t.Fatalf("ConfigureJetStream() resources = %+v", resources)
	}
	if _, err := ConfigureJetStream(connection); err != nil {
		t.Fatalf("ConfigureJetStream() repeat error = %v", err)
	}

	jetStream, err := connection.JetStream()
	if err != nil {
		t.Fatal(err)
	}
	consumer, err := jetStream.ConsumerInfo(activityStreamName, activityConsumerName)
	if err != nil {
		t.Fatal(err)
	}
	if consumer.Config.AckPolicy != natsgo.AckExplicitPolicy || consumer.Config.MaxDeliver != 5 {
		t.Fatalf("consumer configuration = %+v", consumer.Config)
	}
}

func TestConfigureJetStreamRejectsNilConnection(t *testing.T) {
	if _, err := ConfigureJetStream(nil); err == nil {
		t.Fatal("ConfigureJetStream(nil) returned nil error")
	}
}
