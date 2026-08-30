// Package events defines the job lifecycle and event payloads used by the video
// processing pipeline.
package events

import "fmt"

// JobState is the lifecycle stage for a processing job.
type JobState string

const (
	JobPending   JobState = "pending"
	JobQueued    JobState = "queued"
	JobRunning   JobState = "running"
	JobSucceeded JobState = "succeeded"
	JobFailed    JobState = "failed"
	JobCancelled JobState = "cancelled"
)

// IsValid reports whether the state is one of the supported lifecycle states.
func (s JobState) IsValid() bool {
	switch s {
	case JobPending, JobQueued, JobRunning, JobSucceeded, JobFailed, JobCancelled:
		return true
	default:
		return false
	}
}

// IsTerminal reports whether the job has reached a state from which it cannot
// transition to another lifecycle state.
func (s JobState) IsTerminal() bool {
	return s == JobSucceeded || s == JobFailed || s == JobCancelled
}

// CanTransition reports whether next is an allowed successor of s.
func (s JobState) CanTransition(next JobState) bool {
	switch s {
	case JobPending:
		return next == JobQueued
	case JobQueued:
		return next == JobRunning || next == JobCancelled
	case JobRunning:
		return next == JobSucceeded || next == JobFailed
	default:
		return false
	}
}

// ValidateTransition returns an error when a job state change violates the
// lifecycle contract.
func ValidateTransition(current, next JobState) error {
	if !current.IsValid() {
		return fmt.Errorf("invalid current job state %q", current)
	}
	if !next.IsValid() {
		return fmt.Errorf("invalid next job state %q", next)
	}
	if !current.CanTransition(next) {
		return fmt.Errorf("job state cannot transition from %q to %q", current, next)
	}
	return nil
}

// JobCreatedPayload describes the initial event emitted when a job is created.
type JobCreatedPayload struct {
	JobID     string   `json:"job_id"`
	UserID    string   `json:"user_id"`
	SessionID string   `json:"session_id"`
	State     JobState `json:"state"`
}

// JobUpdatedPayload describes the state transition emitted after a job changes.
type JobUpdatedPayload struct {
	JobID         string   `json:"job_id"`
	PreviousState JobState `json:"previous_state,omitempty"`
	CurrentState  JobState `json:"current_state"`
	Message       string   `json:"message,omitempty"`
}

// JobDonePayload describes the terminal state for a completed or failed job.
type JobDonePayload struct {
	JobID     string   `json:"job_id"`
	State     JobState `json:"state"`
	OutputURL string   `json:"output_url,omitempty"`
	Error     string   `json:"error,omitempty"`
}
