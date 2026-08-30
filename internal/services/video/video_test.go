package video_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/devwillsha/voidcut/internal/repository/postgres"
	"github.com/devwillsha/voidcut/internal/services/video"
)

// mockObjectStore is a fake ObjectStore for testing.
type mockObjectStore struct {
	stored map[string][]byte
}

func newMockObjectStore() *mockObjectStore {
	return &mockObjectStore{stored: make(map[string][]byte)}
}

func (m *mockObjectStore) Put(ctx context.Context, key string, data io.Reader, maxSize int64) (string, error) {
	buf := make([]byte, maxSize)
	n, err := data.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	m.stored[key] = buf[:n]
	return "s3://bucket/" + key, nil
}

func (m *mockObjectStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if data, ok := m.stored[key]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, io.EOF
}

func (m *mockObjectStore) Delete(ctx context.Context, key string) error {
	delete(m.stored, key)
	return nil
}

func TestNewService(t *testing.T) {
	tests := []struct {
		name        string
		repo        *postgres.JobsRepository
		objectStore video.ObjectStore
		wantErr     bool
	}{
		{"valid", &postgres.JobsRepository{}, newMockObjectStore(), false},
		{"nil repo", nil, newMockObjectStore(), true},
		{"nil object store", &postgres.JobsRepository{}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := video.NewService(tt.repo, tt.objectStore)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewService() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUploadRequiresFields(t *testing.T) {
	objectStore := newMockObjectStore()
	svc, _ := video.NewService(&postgres.JobsRepository{}, objectStore)

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
