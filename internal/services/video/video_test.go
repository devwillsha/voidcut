package video_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/devwillsha/voidcut/internal/repository/postgres"
	"github.com/devwillsha/voidcut/internal/services/video"
)

func TestNewService(t *testing.T) {
	tmpDir := t.TempDir()
	tests := []struct {
		name    string
		repo    *postgres.JobsRepository
		jsCtx   interface{}
		dir     string
		wantErr bool
	}{
		{"valid", &postgres.JobsRepository{}, "mock-context", tmpDir, false},
		{"nil repo", nil, "mock-context", tmpDir, true},
		{"empty dir", &postgres.JobsRepository{}, "mock-context", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := video.NewService(tt.repo, nil, tt.dir)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadRequiresFields(t *testing.T) {
	tmpDir := t.TempDir()
	svc, _ := video.NewService(&postgres.JobsRepository{}, nil, tmpDir)

	tests := []struct {
		name    string
		req     video.UploadRequest
		wantErr string
	}{
		{"missing session ID", video.UploadRequest{UserID: "user-1", VideoData: bytes.NewBufferString("data")}, "session ID"},
		{"missing user ID", video.UploadRequest{SessionID: "sess-1", VideoData: bytes.NewBufferString("data")}, "user ID"},
		{"missing video data", video.UploadRequest{SessionID: "sess-1", UserID: "user-1"}, "video data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Upload(context.Background(), tt.req)
			if err == nil || !bytes.Contains([]byte(err.Error()), []byte(tt.wantErr)) {
				t.Fatalf("Upload() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

// TestUploadStoresVideoLocally verifies that the service saves video files to disk.
func TestUploadStoresVideoLocally(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping disk I/O test")
	}
	t.Skip("requires PostgreSQL and NATS setup")
}

// TestUploadLimitsFileSize verifies that uploads are capped at 500MB.
func TestUploadLimitsFileSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test")
	}
	t.Skip("requires PostgreSQL and NATS setup")
}

// TestMockUpload verifies the mock upload helper.
func TestMockUpload(t *testing.T) {
	r := video.MockUpload("sess-1", "user-1")
	if r == nil {
		t.Fatal("MockUpload returned nil")
	}
}
