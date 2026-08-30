package events

import "testing"

func TestJobStateTransitions(t *testing.T) {
	valid := map[JobState][]JobState{
		JobPending: {JobQueued},
		JobQueued:  {JobRunning, JobCancelled},
		JobRunning: {JobSucceeded, JobFailed},
	}

	for current, nextStates := range valid {
		for _, next := range nextStates {
			if err := ValidateTransition(current, next); err != nil {
				t.Errorf("ValidateTransition(%q, %q) error = %v", current, next, err)
			}
			if !current.CanTransition(next) {
				t.Errorf("CanTransition(%q, %q) = false", current, next)
			}
		}
	}
}

func TestJobStateTransitionsRejectInvalidChanges(t *testing.T) {
	invalid := [][2]JobState{
		{JobPending, JobRunning},
		{JobQueued, JobSucceeded},
		{JobRunning, JobCancelled},
		{JobSucceeded, JobRunning},
		{JobFailed, JobQueued},
		{JobCancelled, JobQueued},
		{"unknown", JobQueued},
		{JobPending, "unknown"},
	}

	for _, transition := range invalid {
		if err := ValidateTransition(transition[0], transition[1]); err == nil {
			t.Errorf("ValidateTransition(%q, %q) returned nil", transition[0], transition[1])
		}
	}
}

func TestJobStateTerminalValues(t *testing.T) {
	for _, state := range []JobState{JobSucceeded, JobFailed, JobCancelled} {
		if !state.IsTerminal() {
			t.Errorf("IsTerminal(%q) = false", state)
		}
	}
	for _, state := range []JobState{JobPending, JobQueued, JobRunning} {
		if state.IsTerminal() {
			t.Errorf("IsTerminal(%q) = true", state)
		}
	}
}
