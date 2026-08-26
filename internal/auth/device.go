package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// DeviceCodeResponse is returned by the Gateway device-login start endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceLoginClient requests device-login codes from the Gateway.
type DeviceLoginClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// Start requests a device code for the local agent.
func (client DeviceLoginClient) Start(ctx context.Context) (DeviceCodeResponse, error) {
	if strings.TrimSpace(client.BaseURL) == "" {
		return DeviceCodeResponse{}, fmt.Errorf("gateway URL is required")
	}
	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(client.BaseURL, "/")+"/api/v1/auth/device/start", nil)
	if err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("create device start request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("request device code: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return DeviceCodeResponse{}, fmt.Errorf("device start returned HTTP %s", response.Status)
	}

	var deviceCode DeviceCodeResponse
	if err := json.NewDecoder(response.Body).Decode(&deviceCode); err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("decode device start response: %w", err)
	}
	if deviceCode.DeviceCode == "" || deviceCode.UserCode == "" || deviceCode.VerificationURL == "" {
		return DeviceCodeResponse{}, fmt.Errorf("device start response is missing required fields")
	}
	if deviceCode.ExpiresIn <= 0 || deviceCode.Interval <= 0 {
		return DeviceCodeResponse{}, fmt.Errorf("device start response has invalid timing values")
	}
	return deviceCode, nil
}
