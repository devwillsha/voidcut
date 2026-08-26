package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var ErrDeviceLoginExpired = errors.New("device login expired")
var ErrDeviceLoginDenied = errors.New("device login denied")

// DeviceCodeResponse is returned by the Gateway device-login start endpoint.
type DeviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// DeviceTokenResponse is returned by the Gateway device-token polling endpoint.
type DeviceTokenResponse struct {
	Status    string `json:"status"`
	Token     string `json:"token"`
	UserID    string `json:"user_id"`
	ExpiresIn int    `json:"expires_in"`
}

// CredentialsFromDeviceToken converts an approved Gateway response into the
// local credential format used by the agent.
func CredentialsFromDeviceToken(response DeviceTokenResponse, now time.Time) (Credentials, error) {
	if response.Status != "approved" || strings.TrimSpace(response.Token) == "" {
		return Credentials{}, ErrInvalidCredentials
	}
	credentials := Credentials{Token: response.Token, UserID: response.UserID}
	if response.ExpiresIn > 0 {
		credentials.ExpiresAt = now.Add(time.Duration(response.ExpiresIn) * time.Second)
	}
	return credentials, credentials.Validate(now)
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

// Poll waits for device approval and returns the issued token. It polls
// immediately, then waits the Gateway-provided interval between attempts.
func (client DeviceLoginClient) Poll(ctx context.Context, deviceCode DeviceCodeResponse) (DeviceTokenResponse, error) {
	if strings.TrimSpace(deviceCode.DeviceCode) == "" {
		return DeviceTokenResponse{}, fmt.Errorf("device code is required")
	}
	if deviceCode.Interval <= 0 || deviceCode.ExpiresIn <= 0 {
		return DeviceTokenResponse{}, fmt.Errorf("device code has invalid timing values")
	}
	if strings.TrimSpace(client.BaseURL) == "" {
		return DeviceTokenResponse{}, fmt.Errorf("gateway URL is required")
	}

	pollContext, cancel := context.WithTimeout(ctx, time.Duration(deviceCode.ExpiresIn)*time.Second)
	defer cancel()
	deadline := time.NewTimer(time.Duration(deviceCode.ExpiresIn) * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(time.Duration(deviceCode.Interval) * time.Second)
	defer ticker.Stop()

	for attempt := 0; ; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return DeviceTokenResponse{}, ctx.Err()
			case <-deadline.C:
				return DeviceTokenResponse{}, ErrDeviceLoginExpired
			case <-ticker.C:
			}
		}

		response, err := client.pollOnce(pollContext, deviceCode.DeviceCode)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return DeviceTokenResponse{}, ErrDeviceLoginExpired
			}
			return DeviceTokenResponse{}, err
		}
		switch response.Status {
		case "pending":
			continue
		case "approved":
			if strings.TrimSpace(response.Token) == "" {
				return DeviceTokenResponse{}, fmt.Errorf("approved device response is missing token")
			}
			return response, nil
		case "expired":
			return DeviceTokenResponse{}, ErrDeviceLoginExpired
		case "denied":
			return DeviceTokenResponse{}, ErrDeviceLoginDenied
		default:
			return DeviceTokenResponse{}, fmt.Errorf("unknown device token status %q", response.Status)
		}
	}
}

func (client DeviceLoginClient) pollOnce(ctx context.Context, deviceCode string) (DeviceTokenResponse, error) {
	httpClient := client.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	body, err := json.Marshal(struct {
		DeviceCode string `json:"device_code"`
	}{DeviceCode: deviceCode})
	if err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("encode device token request: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(client.BaseURL, "/")+"/api/v1/auth/device/token", strings.NewReader(string(body)))
	if err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("create device token request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	response, err := httpClient.Do(request)
	if err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("poll device token: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return DeviceTokenResponse{}, fmt.Errorf("device token poll returned HTTP %s", response.Status)
	}

	var tokenResponse DeviceTokenResponse
	if err := json.NewDecoder(response.Body).Decode(&tokenResponse); err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("decode device token response: %w", err)
	}
	return tokenResponse, nil
}
