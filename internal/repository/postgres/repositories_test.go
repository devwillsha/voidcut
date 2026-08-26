package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/schema"
)

func TestRepositoriesRejectNilPool(t *testing.T) {
	if _, err := NewActivityRepository(nil); err == nil {
		t.Fatal("NewActivityRepository(nil) returned nil error")
	}
	if _, err := NewJobsRepository(nil); err == nil {
		t.Fatal("NewJobsRepository(nil) returned nil error")
	}
}

func TestActivityInsertValidatesRequiredFieldsBeforeDatabaseCall(t *testing.T) {
	repository := &ActivityRepository{}
	for _, event := range []schema.EventEnvelope{
		{},
		{EventID: "event-1", TraceID: "trace-1", UserID: "user-1", SessionID: "session-1", DeviceID: "device-1", EventType: "keyboard", Version: schema.VersionV1, Timestamp: time.Now()},
	} {
		if err := repository.Insert(context.Background(), event); err == nil {
			t.Fatal("Insert() accepted an invalid event without a pool")
		}
	}
}

func TestJobsValidateBeforeDatabaseCall(t *testing.T) {
	repository := &JobsRepository{}
	if _, err := repository.Create(context.Background(), Job{}); err == nil {
		t.Fatal("Create() accepted an invalid job without a pool")
	}
	if _, err := repository.UpdateState(context.Background(), "job-1", events.JobPending, events.JobRunning); err == nil {
		t.Fatal("UpdateState() accepted an invalid transition")
	}
}
