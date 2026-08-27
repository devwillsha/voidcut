// Package video implements the VideoService for handling video uploads and job
// creation in the Voidcut pipeline.
package video

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/repository/postgres"
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

// Service manages video uploads and job creation.
type Service struct {
	jobsRepo      *postgres.JobsRepository
	objectStore   ObjectStore
	maxUploadSize int64
}

// NewService creates a new VideoService.
func NewService(jobsRepo *postgres.JobsRepository, objectStore ObjectStore) (*Service, error) {
	if jobsRepo == nil {
		return nil, errors.New("jobs repository is required")
	}
	if objectStore == nil {
		return nil, errors.New("object store is required")
	}

	return &Service{
		jobsRepo:      jobsRepo,
		objectStore:   objectStore,
		maxUploadSize: 500 * 1024 * 1024, // 500MB limit
	}, nil
}

// Upload processes a video upload request and creates a job record.
// Returns immediately after storing video and creating the job record.
// NATS publishing is handled separately (Task 27).
func (s *Service) Upload(ctx context.Context, req UploadRequest) (UploadResponse, error) {
	if s == nil || s.jobsRepo == nil || s.objectStore == nil {
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

	// Wrap video data to enforce size limit before storing.
	limitedData := &io.LimitedReader{
		R: req.VideoData,
		N: s.maxUploadSize,
	}

	// Store video using ObjectStore (S3-compatible or local).
	objectKey := jobID + ".mp4"
	inputURL, err := s.objectStore.Put(ctx, objectKey, limitedData, s.maxUploadSize)
	if err != nil {
		return UploadResponse{}, fmt.Errorf("store video: %w", err)
	}

	// Verify that we didn't exceed the size limit.
	if limitedData.N < 0 {
		// Best-effort cleanup.
		_ = s.objectStore.Delete(ctx, objectKey)
		return UploadResponse{}, errors.New("video exceeds maximum upload size of 500MB")
	}

	// Create job record in database.
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
		_ = s.objectStore.Delete(ctx, objectKey)
		return UploadResponse{}, fmt.Errorf("create job record: %w", err)
	}

	return UploadResponse{
		JobID:     createdJob.JobID,
		State:     string(createdJob.State),
		InputURL:  inputURL,
		CreatedAt: createdJob.CreatedAt.Format(time.RFC3339),
	}, nil
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
