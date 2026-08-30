package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Job is the database representation of a video-processing job.
type Job struct {
	JobID     string
	TraceID   string
	UserID    string
	SessionID string
	State     events.JobState
	InputURL  *string
	OutputURL *string
	Error     *string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// JobsRepository persists video-processing jobs.
type JobsRepository struct {
	pool *pgxpool.Pool
}

// NewJobsRepository creates a jobs repository backed by pool.
func NewJobsRepository(pool *pgxpool.Pool) (*JobsRepository, error) {
	if pool == nil {
		return nil, errNilPool
	}
	return &JobsRepository{pool: pool}, nil
}

// Create inserts a job and returns the database timestamps.
func (repository *JobsRepository) Create(ctx context.Context, job Job) (Job, error) {
	if job.JobID == "" || job.TraceID == "" || job.UserID == "" || job.SessionID == "" {
		return Job{}, errors.New("job is missing a required field")
	}
	if job.State == "" {
		job.State = events.JobPending
	}
	if !job.State.IsValid() {
		return Job{}, fmt.Errorf("invalid job state %q", job.State)
	}

	const query = `
		INSERT INTO jobs (
			job_id, trace_id, user_id, session_id, state, input_url, output_url, error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	err := repository.pool.QueryRow(ctx, query,
		job.JobID,
		job.TraceID,
		job.UserID,
		job.SessionID,
		job.State,
		job.InputURL,
		job.OutputURL,
		job.Error,
	).Scan(&job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return Job{}, err
	}
	return job, nil
}

// GetByID returns a job by its unique identifier.
func (repository *JobsRepository) GetByID(ctx context.Context, jobID string) (Job, error) {
	if jobID == "" {
		return Job{}, errors.New("job ID is required")
	}

	const query = `
		SELECT job_id, trace_id, user_id, session_id, state,
			input_url, output_url, error, created_at, updated_at
		FROM jobs
		WHERE job_id = $1`

	return scanJob(repository.pool.QueryRow(ctx, query, jobID))
}

// UpdateState advances a job through the Task 2 lifecycle contract. The
// expected current state is included in the WHERE clause to prevent a stale
// worker from overwriting a concurrent state change.
func (repository *JobsRepository) UpdateState(ctx context.Context, jobID string, current, next events.JobState) (Job, error) {
	if jobID == "" {
		return Job{}, errors.New("job ID is required")
	}
	if err := events.ValidateTransition(current, next); err != nil {
		return Job{}, err
	}

	const query = `
		UPDATE jobs
		SET state = $1, updated_at = NOW()
		WHERE job_id = $2 AND state = $3
		RETURNING job_id, trace_id, user_id, session_id, state,
			input_url, output_url, error, created_at, updated_at`

	return scanJob(repository.pool.QueryRow(ctx, query, next, jobID, current))
}

func scanJob(row pgx.Row) (Job, error) {
	var job Job
	if err := row.Scan(
		&job.JobID,
		&job.TraceID,
		&job.UserID,
		&job.SessionID,
		&job.State,
		&job.InputURL,
		&job.OutputURL,
		&job.Error,
		&job.CreatedAt,
		&job.UpdatedAt,
	); err != nil {
		return Job{}, err
	}
	return job, nil
}
