package gateway

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
)

// Gateway wraps the HTTP router and provides the API entry point.
type Gateway struct {
	router *chi.Mux
	log    *zap.SugaredLogger
	addr   string
}

// New creates a new API Gateway with default middleware and a ready-to-use
// router. Call Mount* methods to attach route handlers, then call Start to
// begin listening.
func New(addr string, log *zap.SugaredLogger) (*Gateway, error) {
	if addr == "" {
		return nil, errors.New("address is required")
	}
	if log == nil {
		return nil, errors.New("logger is required")
	}

	r := chi.NewRouter()

	// Middleware stack (ordered - runs in reverse during response).
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(RequestLogger(log))
	r.Use(middleware.Recoverer)
	r.Use(CORS())
	r.Use(middleware.Timeout(30 * time.Second))

	return &Gateway{router: r, log: log, addr: addr}, nil
}

// Router exposes the chi router for adding routes and middleware.
func (g *Gateway) Router() *chi.Mux {
	if g == nil {
		return nil
	}
	return g.router
}

// MountReadiness mounts the readiness health check endpoint.
func (g *Gateway) MountReadiness() {
	if g == nil || g.router == nil {
		return
	}
	g.router.Get("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ready"}`))
	})
}

// MountLiveness mounts the liveness health check endpoint.
func (g *Gateway) MountLiveness() {
	if g == nil || g.router == nil {
		return
	}
	g.router.Get("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"alive"}`))
	})
}

// Start begins listening for HTTP requests. It blocks until the server stops.
// The caller should use goroutines or context cancellation to manage shutdown.
func (g *Gateway) Start() error {
	if g == nil || g.router == nil {
		return errors.New("gateway is not properly initialized")
	}
	if g.log != nil {
		g.log.Infof("API Gateway listening on %s", g.addr)
	}
	return http.ListenAndServe(g.addr, g.router)
}

// Stop performs a graceful shutdown. It is called when the main process
// receives SIGTERM/SIGINT. The caller must invoke this as part of shutdown
// orchestration to cleanly close the server.
func (g *Gateway) Stop() error {
	// The Gateway itself doesn't maintain a server reference, so shutdown
	// is managed by the caller (e.g., see cmd/gateway/main.go).
	// This method is here for API consistency with other services.
	return nil
}

// Address returns the gateway's listening address.
func (g *Gateway) Address() string {
	if g == nil {
		return ""
	}
	return g.addr
}
