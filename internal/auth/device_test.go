package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
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
