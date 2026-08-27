package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/devwillsha/voidcut/internal/config"
	"github.com/devwillsha/voidcut/internal/logging"
	"github.com/devwillsha/voidcut/internal/repository/postgres"
	"github.com/devwillsha/voidcut/internal/services/video"
	natsgo "github.com/nats-io/nats.go"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	cfg := config.Load()

	logger, err := logging.New("video")
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	sugar := logger.Sugar()

	// Connect to PostgreSQL.
	pgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := pgxpool.New(pgCtx, cfg.PostgresDSN)
	cancel()
	if err != nil {
		sugar.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	// Connect to NATS.
	nc, err := natsgo.Connect(cfg.NATSURL)
	if err != nil {
		sugar.Fatalf("connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream context for publishing job.created events.
	js, err := nc.JetStream()
	if err != nil {
		sugar.Fatalf("create JetStream context: %v", err)
	}

	// Create repositories and services.
	jobsRepo, err := postgres.NewJobsRepository(pool)
	if err != nil {
		sugar.Fatalf("create jobs repository: %v", err)
	}

	uploadDir := os.Getenv("VOIDCUT_VIDEO_UPLOAD_DIR")
	if uploadDir == "" {
		uploadDir = "./videos"
	}

	videoSvc, err := video.NewService(jobsRepo, js, uploadDir)
	if err != nil {
		sugar.Fatalf("create video service: %v", err)
	}

	// Setup HTTP server.
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/jobs", video.UploadHandler(videoSvc))
	mux.HandleFunc("GET /health/ready", healthReadyHandler)
	mux.HandleFunc("GET /health/live", healthLiveHandler)

	server := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	sugar.Infow("starting VideoService", "addr", server.Addr, "uploadDir", uploadDir)

	// Start server in background.
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			sugar.Errorw("server error", "err", err)
		}
	}()

	// Wait for shutdown signal.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan

	// Graceful shutdown.
	sugar.Infow("shutting down VideoService")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		sugar.Errorw("shutdown error", "err", err)
		os.Exit(1)
	}

	sugar.Infow("VideoService stopped")
}

func healthReadyHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ready"}`)
}

func healthLiveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"alive"}`)
}
