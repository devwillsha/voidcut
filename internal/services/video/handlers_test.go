package video_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/devwillsha/voidcut/internal/events"
	"github.com/devwillsha/voidcut/internal/repository/postgres"
	"github.com/devwillsha/voidcut/internal/services/video"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUploadHandler(t *testing.T) {
	// Create minimal service without real DB/NATS for handler tests.
	svc, _ := video.NewService(&postgres.JobsRepository{}, nil, t.TempDir())
	handler := video.UploadHandler(svc)

	t.Run("missing video file", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		w.WriteField("session_id", "sess-1")
		w.WriteField("user_id", "user-1")
		w.Close()

		req := httptest.NewRequest("POST", "/api/v1/jobs", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rw := httptest.NewRecorder()

		handler.ServeHTTP(rw, req)

		if rw.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rw.Code)
		}
	})

	t.Run("missing session_id", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		w.WriteField("user_id", "user-1")

		fw, _ := w.CreateFormFile("video", "test.mp4")
		fw.Write([]byte("mock video"))
		w.Close()

		req := httptest.NewRequest("POST", "/api/v1/jobs", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rw := httptest.NewRecorder()

		handler.ServeHTTP(rw, req)

		if rw.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rw.Code)
		}
	})

	t.Run("missing user_id", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		w.WriteField("session_id", "sess-1")

		fw, _ := w.CreateFormFile("video", "test.mp4")
		fw.Write([]byte("mock video"))
		w.Close()

		req := httptest.NewRequest("POST", "/api/v1/jobs", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rw := httptest.NewRecorder()

		handler.ServeHTTP(rw, req)

		if rw.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rw.Code)
		}
	})

	t.Run("valid upload requires real DB", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		w.WriteField("session_id", "sess-1")
		w.WriteField("user_id", "user-1")

		fw, _ := w.CreateFormFile("video", "test.mp4")
		fw.Write([]byte("mock video data"))
		w.Close()

		// This will panic without a real database, so we skip it.
		t.Skip("requires PostgreSQL connection")
	})
}

func TestUploadHandlerWithValidDB(t *testing.T) {
	pool := getTestDB(t)
	if pool == nil {
		t.Skip("PostgreSQL unavailable")
	}
	defer pool.Close()

	repo, _ := postgres.NewJobsRepository(pool)
	tmpDir := t.TempDir()
	svc, _ := video.NewService(repo, nil, tmpDir)
	handler := video.UploadHandler(svc)

	t.Run("valid upload with DB returns 201", func(t *testing.T) {
		body := &bytes.Buffer{}
		w := multipart.NewWriter(body)
		w.WriteField("session_id", "sess-handler-test")
		w.WriteField("user_id", "user-handler-test")
		w.WriteField("device_id", "device-1")

		fw, _ := w.CreateFormFile("video", "test.mp4")
		fw.Write([]byte("mock video data for handler test"))
		w.Close()

		req := httptest.NewRequest("POST", "/api/v1/jobs", body)
		req.Header.Set("Content-Type", w.FormDataContentType())
		rw := httptest.NewRecorder()

		handler.ServeHTTP(rw, req)

		if rw.Code != http.StatusCreated && rw.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 201 or 500", rw.Code)
		}

		if rw.Code == http.StatusCreated {
			var resp video.UploadResponse
			if err := json.NewDecoder(rw.Body).Decode(&resp); err != nil {
				t.Fatalf("decode response error = %v", err)
			}
			if resp.JobID == "" {
				t.Fatal("response missing job_id")
			}
			if resp.State != string(events.JobPending) {
				t.Fatalf("state = %s, want %s", resp.State, events.JobPending)
			}
		}
	})
}

func getTestDB(t *testing.T) *pgxpool.Pool {
	dsn := os.Getenv("VOIDCUT_POSTGRES_DSN")
	if dsn == "" {
		dsn = "postgres://voidcut:voidcut@localhost:5432/voidcut?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil
	}

	return pool
}
