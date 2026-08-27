// Package video implements the VideoService for handling video uploads and job
// creation in the Voidcut pipeline.
package video

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/repository/postgres"
	"github.com/devwillsha/voidcut/internal/schema"
	natsgo "github.com/nats-io/nats.go"
)

// UploadRequest contains the metadata for a video upload.
type UploadRequest struct {
	SessionID string
	UserID    string
	DeviceID  string
	VideoData io.Reader
}

// UploadResponse describes the result of a successful upload.
type UploadResponse struct {
	JobID     string `json:"job_id"`
	State     string `json:"state"`
	InputURL  string `json:"input_url"`
	CreatedAt string `json:"created_at"`
}

// Service manages video uploads and job lifecycle.
type Service struct {
	jobsRepo  *postgres.JobsRepository
	js        natsgo.JetStreamContext
	uploadDir string
}

// NewService creates a new VideoService.
func NewService(jobsRepo *postgres.JobsRepository, jsCtx natsgo.JetStreamContext, uploadDir string) (*Service, error) {
	if jobsRepo == nil {
		return nil, errors.New("jobs repository is required")
	}
	// jsCtx can be nil for testing; NATS publishing will be skipped.
	if uploadDir == "" {
		return nil, errors.New("upload directory is required")
	}

	// Ensure upload directory exists.
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("create upload directory: %w", err)
	}

	return &Service{
		jobsRepo:  jobsRepo,
		js:        jsCtx,
		uploadDir: uploadDir,
	}, nil
}

// Upload processes a video upload request and creates a job record.
func (s *Service) Upload(ctx context.Context, req UploadRequest) (UploadResponse, error) {
	if s == nil || s.jobsRepo == nil {
		return UploadResponse{}, errors.New("service is not properly initialized")
	}
	if req.SessionID == "" {
		return UploadResponse{}, errors.New("session ID is required")
	}
	if req.UserID == "" {
		return UploadResponse{}, errors.New("user ID is required")
	}
	if req.VideoData == nil {
		return UploadResponse{}, errors.New("video data is required")
	}

	// Generate IDs.
	jobID := generateID(16)
	traceID := generateID(16)

	// Read and store video file.
	videoPath := filepath.Join(s.uploadDir, jobID+".mp4")
	if err := s.storeVideo(videoPath, req.VideoData); err != nil {
		return UploadResponse{}, fmt.Errorf("store video: %w", err)
	}

	// Create job record in database.
	inputURL := "file://" + videoPath
	job := postgres.Job{
		JobID:     jobID,
		TraceID:   traceID,
		UserID:    req.UserID,
		SessionID: req.SessionID,
		State:     events.JobPending,
		InputURL:  &inputURL,
	}

	createdJob, err := s.jobsRepo.Create(ctx, job)
	if err != nil {
		// Best-effort cleanup.
		_ = os.Remove(videoPath)
		return UploadResponse{}, fmt.Errorf("create job record: %w", err)
	}

	// Publish job.created event to NATS.
	if err := s.publishJobCreatedEvent(ctx, createdJob, req.DeviceID); err != nil {
		// Log but don't fail the upload; the job record exists.
		fmt.Fprintf(os.Stderr, "failed to publish job.created event: %v\n", err)
	}

	return UploadResponse{
		JobID:     createdJob.JobID,
		State:     string(createdJob.State),
		InputURL:  inputURL,
		CreatedAt: createdJob.CreatedAt.Format(time.RFC3339),
	}, nil
}

// storeVideo writes video data to disk.
func (s *Service) storeVideo(path string, data io.Reader) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Limit read to 500MB to prevent DoS.
	limitedReader := io.LimitReader(data, 500*1024*1024)
	if _, err := io.Copy(file, limitedReader); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// publishJobCreatedEvent publishes a job.created event to NATS.
func (s *Service) publishJobCreatedEvent(ctx context.Context, job postgres.Job, deviceID string) error {
	payload := events.JobCreatedPayload{
		JobID:     job.JobID,
		UserID:    job.UserID,
		SessionID: job.SessionID,
		State:     job.State,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	envelope := schema.EventEnvelope{
		EventID:   generateID(16),
		TraceID:   job.TraceID,
		UserID:    job.UserID,
		SessionID: job.SessionID,
		DeviceID:  deviceID,
		EventType: "job.created",
		Version:   schema.VersionV1,
		Timestamp: time.Now().UTC(),
		Payload:   payloadBytes,
	}

	envelopeBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	_, err = s.js.Publish(string(schema.JobCreatedV1), envelopeBytes)
	if err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	return nil
}

// generateID creates a random hex-encoded ID of the specified byte length.
func generateID(byteLength int) string {
	b := make([]byte, byteLength)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

// MockUpload is a test helper that returns a simple test video payload.
func MockUpload(sessionID, userID string) io.Reader {
	return bytes.NewBufferString("mock video data")
}
