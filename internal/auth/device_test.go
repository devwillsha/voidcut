package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDeviceLoginClientStart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", request.Method)
		}
		if request.URL.Path != "/api/v1/auth/device/start" {
			t.Errorf("path = %s, want /api/v1/auth/device/start", request.URL.Path)
		}
		if request.Header.Get("Accept") != "application/json" {
			t.Errorf("Accept = %q, want application/json", request.Header.Get("Accept"))
		}
		writer.Header().Set("Content-Type", "application/json")
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{"device_code":"device-1","user_code":"ABCD-1234","verification_url":"https://voidcut.app/connect?code=ABCD-1234","expires_in":600,"interval":5}`))
	}))
	defer server.Close()

	response, err := (DeviceLoginClient{BaseURL: server.URL}).Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if response.DeviceCode != "device-1" || response.UserCode != "ABCD-1234" || response.ExpiresIn != 600 || response.Interval != 5 {
		t.Fatalf("unexpected device response: %+v", response)
	}
}

func TestDeviceLoginClientRejectsBadResponses(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{name: "server error", body: `{"error":"unavailable"}`, code: http.StatusBadGateway},
		{name: "malformed JSON", body: `{`, code: http.StatusOK},
		{name: "missing fields", body: `{"device_code":"device-1"}`, code: http.StatusOK},
		{name: "invalid timing", body: `{"device_code":"device-1","user_code":"ABCD-1234","verification_url":"https://voidcut.app/connect","expires_in":0,"interval":5}`, code: http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(test.code)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()

			if _, err := (DeviceLoginClient{BaseURL: server.URL}).Start(context.Background()); err == nil {
				t.Fatal("Start() returned nil error")
			}
		})
	}
}

func TestDeviceLoginClientRequiresGatewayURL(t *testing.T) {
	if _, err := (DeviceLoginClient{}).Start(context.Background()); err == nil {
		t.Fatal("Start() returned nil error for empty gateway URL")
	}
}

func TestDeviceLoginClientPollsUntilApproved(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/v1/auth/device/token" || request.Method != http.MethodPost {
			t.Errorf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		var body struct {
			DeviceCode string `json:"device_code"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body.DeviceCode != "device-1" {
			t.Errorf("device code = %q, want device-1", body.DeviceCode)
		}
		if attempts.Add(1) == 1 {
			_, _ = writer.Write([]byte(`{"status":"pending"}`))
			return
		}
		_, _ = writer.Write([]byte(`{"status":"approved","token":"token-1","user_id":"user-1","expires_in":3600}`))
	}))
	defer server.Close()

	response, err := (DeviceLoginClient{BaseURL: server.URL}).Poll(context.Background(), DeviceCodeResponse{DeviceCode: "device-1", Interval: 1, ExpiresIn: 5})
	if err != nil {
		t.Fatalf("Poll() error = %v", err)
	}
	if response.Status != "approved" || response.Token != "token-1" || attempts.Load() != 2 {
		t.Fatalf("unexpected poll result: %+v after %d attempts", response, attempts.Load())
	}
}

func TestDeviceLoginClientPollStopsOnCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte(`{"status":"pending"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (DeviceLoginClient{BaseURL: server.URL}).Poll(ctx, DeviceCodeResponse{DeviceCode: "device-1", Interval: 1, ExpiresIn: 5})
	if err == nil {
		t.Fatal("Poll() returned nil error after cancellation")
	}
}

func TestDeviceLoginClientPollRejectsTerminalErrors(t *testing.T) {
	for status, expected := range map[string]error{
		"expired": ErrDeviceLoginExpired,
		"denied":  ErrDeviceLoginDenied,
	} {
		t.Run(status, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				_, _ = writer.Write([]byte(`{"status":"` + status + `"}`))
			}))
			defer server.Close()
			_, err := (DeviceLoginClient{BaseURL: server.URL}).Poll(context.Background(), DeviceCodeResponse{DeviceCode: "device-1", Interval: 1, ExpiresIn: 5})
			if !errors.Is(err, expected) {
				t.Fatalf("Poll() error = %v, want %v", err, expected)
			}
		})
	}
}

func TestCredentialsFromDeviceToken(t *testing.T) {
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	credentials, err := CredentialsFromDeviceToken(DeviceTokenResponse{
		Status:    "approved",
		Token:     "token-1",
		UserID:    "user-1",
		ExpiresIn: 3600,
	}, now)
	if err != nil {
		t.Fatalf("CredentialsFromDeviceToken() error = %v", err)
	}
	if credentials.Token != "token-1" || credentials.UserID != "user-1" || !credentials.ExpiresAt.Equal(now.Add(time.Hour)) {
		t.Fatalf("unexpected credentials: %+v", credentials)
	}
}

func TestCredentialsFromDeviceTokenRejectsUnapprovedResponse(t *testing.T) {
	if _, err := CredentialsFromDeviceToken(DeviceTokenResponse{Status: "pending", Token: "token"}, time.Now()); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("CredentialsFromDeviceToken() error = %v, want ErrInvalidCredentials", err)
	}
}
