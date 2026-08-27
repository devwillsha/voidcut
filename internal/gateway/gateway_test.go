package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestNewGateway(t *testing.T) {
	log := zap.NewNop().Sugar()

	tests := []struct {
		name    string
		addr    string
		log     *zap.SugaredLogger
		wantErr bool
	}{
		{"valid", ":8080", log, false},
		{"missing addr", "", log, true},
		{"nil logger", ":8080", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(tt.addr, tt.log)
			if (err != nil) != tt.wantErr {
				t.Fatalf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatal("New() returned nil gateway")
			}
		})
	}
}

func TestGatewayRouter(t *testing.T) {
	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	router := gw.Router()
	if router == nil {
		t.Fatal("Router() returned nil")
	}
}

func TestGatewayAddress(t *testing.T) {
	log := zap.NewNop().Sugar()
	addr := ":9000"
	gw, err := New(addr, log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if gw.Address() != addr {
		t.Fatalf("Address() = %s, want %s", gw.Address(), addr)
	}
}

func TestMountReadiness(t *testing.T) {
	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	gw.MountReadiness()

	req, err := http.NewRequest("GET", "/health/ready", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != `{"status":"ready"}` {
		t.Fatalf("body = %s, want ready", body)
	}
}

func TestMountLiveness(t *testing.T) {
	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	gw.MountLiveness()

	req, err := http.NewRequest("GET", "/health/live", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if body := w.Body.String(); body != `{"status":"alive"}` {
		t.Fatalf("body = %s, want alive", body)
	}
}

func TestCORSMiddleware(t *testing.T) {
	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	gw.router.Get("/test", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Test CORS preflight OPTIONS request.
	req, err := http.NewRequest("OPTIONS", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")

	w := httptest.NewRecorder()
	gw.router.ServeHTTP(w, req)

	// OPTIONS preflight should be allowed by chi/cors.
	if allow := w.Header().Get("Access-Control-Allow-Methods"); allow == "" {
		t.Fatal("CORS headers not set")
	}

	// Test actual request with CORS origin.
	req2, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		t.Fatal(err)
	}
	req2.Header.Set("Origin", "http://localhost:3000")

	w2 := httptest.NewRecorder()
	gw.router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w2.Code, http.StatusOK)
	}
}
