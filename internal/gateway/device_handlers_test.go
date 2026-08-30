package gateway

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/devwillsha/voidcut/internal/auth"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func TestDeviceStartHandler(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rc.Close()

	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	deviceSvc, err := auth.NewDeviceLoginService(rc)
	if err != nil {
		t.Skipf("device service unavailable: %v", err)
	}

	handler := gw.DeviceStartHandler(deviceSvc)

	req, err := http.NewRequest("POST", "/api/v1/auth/device/start", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp auth.DeviceCodeResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error = %v", err)
	}

	if resp.DeviceCode == "" || resp.UserCode == "" {
		t.Fatal("response missing device_code or user_code")
	}
}

func TestDeviceApproveHandler(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rc.Close()

	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	deviceSvc, err := auth.NewDeviceLoginService(rc)
	if err != nil {
		t.Skipf("device service unavailable: %v", err)
	}

	// Start a device code first.
	startResp, err := deviceSvc.Start(httptest.NewRequest("POST", "/", nil).Context(), auth.DeviceStartRequest{})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	approveReq := auth.DeviceApproveRequest{
		DeviceCode: startResp.DeviceCode,
		UserID:     "user-123",
	}
	body, err := json.Marshal(approveReq)
	if err != nil {
		t.Fatal(err)
	}

	handler := gw.DeviceApproveHandler(deviceSvc)
	req, err := http.NewRequest("POST", "/api/v1/auth/device/approve", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestDeviceTokenHandler(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rc.Close()

	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	deviceSvc, err := auth.NewDeviceLoginService(rc)
	if err != nil {
		t.Skipf("device service unavailable: %v", err)
	}

	// Start and approve.
	startResp, err := deviceSvc.Start(httptest.NewRequest("POST", "/", nil).Context(), auth.DeviceStartRequest{})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	approveReq := auth.DeviceApproveRequest{
		DeviceCode: startResp.DeviceCode,
		UserID:     "user-123",
	}
	err = deviceSvc.Approve(httptest.NewRequest("POST", "/", nil).Context(), approveReq)
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	tokenReq := auth.DeviceTokenRequest{DeviceCode: startResp.DeviceCode}
	body, err := json.Marshal(tokenReq)
	if err != nil {
		t.Fatal(err)
	}

	handler := gw.DeviceTokenHandler(deviceSvc)
	req, err := http.NewRequest("POST", "/api/v1/auth/device/token", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	var resp auth.DeviceTokenResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response error = %v", err)
	}

	if resp.Token == "" {
		t.Fatal("response missing token")
	}
}

func TestConnectHandler(t *testing.T) {
	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	handler := gw.ConnectHandler()

	req, err := http.NewRequest("GET", "/connect?user_code=ABC123", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	if w.Header().Get("Content-Type") != "text/html" {
		t.Fatalf("content type = %s, want text/html", w.Header().Get("Content-Type"))
	}

	if !bytes.Contains(w.Body.Bytes(), []byte("ABC123")) {
		t.Fatal("response does not contain user_code")
	}
}

func TestMountDeviceAuthRoutes(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rc.Close()

	log := zap.NewNop().Sugar()
	gw, err := New(":8080", log)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	deviceSvc, err := auth.NewDeviceLoginService(rc)
	if err != nil {
		t.Skipf("device service unavailable: %v", err)
	}

	gw.MountDeviceAuthRoutes(deviceSvc)

	// Test that routes are registered by making requests.
	req, err := http.NewRequest("POST", "/api/v1/auth/device/start", bytes.NewReader([]byte("{}")))
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	gw.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}
