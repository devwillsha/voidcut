package video

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ObjectStore defines the interface for storing video files.
// Implementations can use S3, MinIO, local disk, etc.
type ObjectStore interface {
	// Put stores a video object and returns its key/URL.
	Put(ctx context.Context, key string, data io.Reader, maxSize int64) (url string, err error)
	// Get retrieves a video object.
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes a video object.
	Delete(ctx context.Context, key string) error
}

// LocalObjectStore implements ObjectStore using the local filesystem.
type LocalObjectStore struct {
	basePath string
}

// NewLocalObjectStore creates a local filesystem object store.
func NewLocalObjectStore(basePath string) (*LocalObjectStore, error) {
	if basePath == "" {
		return nil, errors.New("base path is required")
	}

	// Ensure directory exists.
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("create base directory: %w", err)
	}

	return &LocalObjectStore{basePath: basePath}, nil
}

// Put stores a video file locally with size limit enforcement.
func (s *LocalObjectStore) Put(ctx context.Context, key string, data io.Reader, maxSize int64) (string, error) {
	if key == "" {
		return "", errors.New("object key is required")
	}
	if data == nil {
		return "", errors.New("data is required")
	}

	filePath := filepath.Join(s.basePath, filepath.Base(key))

	// Create file with restricted permissions.
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	// Copy data with size limit. If limit is exceeded, the writer returns
	// an error and we clean up.
	limitedReader := io.LimitedReader{R: data, N: maxSize}

	_, err = io.Copy(file, &limitedReader)
	if err != nil {
		_ = os.Remove(filePath)
		return "", fmt.Errorf("write file: %w", err)
	}

	// Verify that we did not exceed the limit (N < 0 means we read more than allowed).
	if limitedReader.N < 0 {
		_ = os.Remove(filePath)
		return "", errors.New("video exceeds maximum upload size")
	}

	// Return the file URL (local path in file:// format).
	fileURL := "file://" + filePath
	return fileURL, nil
}

// Get retrieves a video file locally.
func (s *LocalObjectStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if key == "" {
		return nil, errors.New("object key is required")
	}

	filePath := filepath.Join(s.basePath, filepath.Base(key))
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

// Delete removes a video file locally.
func (s *LocalObjectStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return errors.New("object key is required")
	}

	filePath := filepath.Join(s.basePath, filepath.Base(key))
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete file: %w", err)
	}

	return nil
}
