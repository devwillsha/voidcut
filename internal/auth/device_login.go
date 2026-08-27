package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DeviceLoginService manages the OAuth 2.0 device flow state machine.
// It generates device codes, stores pending approvals, and validates token requests.
type DeviceLoginService struct {
	redis *redis.Client
}

// DeviceStartRequest is sent by the agent to initiate device login.
type DeviceStartRequest struct {
	// Empty for now - can be extended with device info, app version, etc.
}

// DeviceApproveRequest is sent by the browser/user to approve a device code.
type DeviceApproveRequest struct {
	DeviceCode string `json:"device_code"`
	UserID     string `json:"user_id"`
}

// DeviceTokenRequest is sent by the agent to poll for approval.
type DeviceTokenRequest struct {
	DeviceCode string `json:"device_code"`
}

// NewDeviceLoginService creates a device login service backed by Redis.
func NewDeviceLoginService(r *redis.Client) (*DeviceLoginService, error) {
	if r == nil {
		return nil, errors.New("redis client is required")
	}
	// Verify Redis is reachable.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &DeviceLoginService{redis: r}, nil
}

// Start initiates a device login flow. Returns device_code, user_code, and
// verification URL. The device_code is used for polling; the user_code is
// shown to the user for manual entry.
func (s *DeviceLoginService) Start(ctx context.Context, req DeviceStartRequest) (DeviceCodeResponse, error) {
	if s == nil || s.redis == nil {
		return DeviceCodeResponse{}, errors.New("service not initialized")
	}

	deviceCode := generateCode(32)
	userCode := generateCode(8)
	expiry := 10 * time.Minute

	// Store pending device code in Redis with TTL. Format:
	// device:<device_code> = <user_code>:<approval_status>
	// approval_status: "pending" or "approved:<user_id>:<token>"
	key := "device:" + deviceCode
	value := userCode + ":pending"

	if err := s.redis.Set(ctx, key, value, expiry).Err(); err != nil {
		return DeviceCodeResponse{}, fmt.Errorf("redis set failed: %w", err)
	}

	return DeviceCodeResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationURL: "http://localhost:8080/connect",
		ExpiresIn:       int(expiry.Seconds()),
		Interval:        5, // Agent should poll every 5 seconds.
	}, nil
}

// Approve marks a device code as approved by a user. This is called when the
// user approves the login via the browser or approval endpoint.
func (s *DeviceLoginService) Approve(ctx context.Context, req DeviceApproveRequest) error {
	if s == nil || s.redis == nil {
		return errors.New("service not initialized")
	}
	if req.DeviceCode == "" || req.UserID == "" {
		return errors.New("device_code and user_id are required")
	}

	// Generate a token (in production, sign a JWT or issue a session token).
	token := generateCode(32)

	// Fetch current value and verify it's pending.
	key := "device:" + req.DeviceCode
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return errors.New("device code not found or expired")
	}
	if err != nil {
		return fmt.Errorf("redis get failed: %w", err)
	}

	// Parse value: userCode:status.
	// Only proceed if status is "pending".
	parts := splitColon(val)
	if len(parts) < 2 || parts[1] != "pending" {
		return errors.New("device code is not in pending state")
	}

	// Update value to approved state: userCode:approved:userId:token
	approved := parts[0] + ":approved:" + req.UserID + ":" + token
	ttl := 5 * time.Minute

	if err := s.redis.Set(ctx, key, approved, ttl).Err(); err != nil {
		return fmt.Errorf("redis set failed: %w", err)
	}

	return nil
}

// Token returns the access token if the device code has been approved.
// If still pending, returns an error indicating the user should retry.
func (s *DeviceLoginService) Token(ctx context.Context, req DeviceTokenRequest) (DeviceTokenResponse, error) {
	if s == nil || s.redis == nil {
		return DeviceTokenResponse{}, errors.New("service not initialized")
	}
	if req.DeviceCode == "" {
		return DeviceTokenResponse{}, errors.New("device_code is required")
	}

	key := "device:" + req.DeviceCode
	val, err := s.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return DeviceTokenResponse{}, errors.New("device code not found or expired")
	}
	if err != nil {
		return DeviceTokenResponse{}, fmt.Errorf("redis get failed: %w", err)
	}

	// Parse value: userCode:status[:userID[:token]]
	parts := splitColon(val)
	if len(parts) < 2 {
		return DeviceTokenResponse{}, errors.New("invalid device code format")
	}

	if parts[1] != "approved" || len(parts) < 4 {
		// Not approved yet.
		return DeviceTokenResponse{}, errors.New("authorization_pending")
	}

	// Return token.
	token := parts[3]
	userID := parts[2]
	return DeviceTokenResponse{
		Status:    "approved",
		Token:     token,
		UserID:    userID,
		ExpiresIn: 3600,
	}, nil
}

func generateCode(length int) string {
	buf := make([]byte, length)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)[:length]
}

func splitColon(s string) []string {
	return splitString(s, ':')
}

func splitString(s string, sep byte) []string {
	var parts []string
	var current []byte
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, string(current))
			current = current[:0]
		} else {
			current = append(current, s[i])
		}
	}
	parts = append(parts, string(current))
	return parts
}
