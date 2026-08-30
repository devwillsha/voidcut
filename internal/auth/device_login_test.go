package auth

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNewDeviceLoginService(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	defer rc.Close()

	// Check if Redis is available.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	redisAvailable := rc.Ping(ctx).Err() == nil

	tests := []struct {
		name    string
		redis   *redis.Client
		wantErr bool
	}{
		{"valid", rc, false},
		{"nil redis", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip valid test if Redis unavailable.
			if tt.name == "valid" && !redisAvailable {
				t.Skipf("Redis unavailable")
			}

			got, err := NewDeviceLoginService(tt.redis)
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewDeviceLoginService() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got == nil {
				t.Fatal("NewDeviceLoginService() returned nil service")
			}
		})
	}
}

func TestStart(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Skip if Redis unavailable.
	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	defer rc.Close()

	svc, err := NewDeviceLoginService(rc)
	if err != nil {
		t.Fatalf("NewDeviceLoginService() error = %v", err)
	}

	resp, err := svc.Start(ctx, DeviceStartRequest{})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if resp.DeviceCode == "" || resp.UserCode == "" {
		t.Fatal("Start() returned empty codes")
	}
	if resp.ExpiresIn != 600 {
		t.Fatalf("ExpiresIn = %d, want 600", resp.ExpiresIn)
	}

	// Verify device code is stored in Redis.
	key := "device:" + resp.DeviceCode
	val, err := rc.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("redis get failed: %v", err)
	}
	if val != resp.UserCode+":pending" {
		t.Fatalf("stored value = %s, want %s:pending", val, resp.UserCode)
	}
}

func TestApprove(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	defer rc.Close()

	svc, err := NewDeviceLoginService(rc)
	if err != nil {
		t.Fatalf("NewDeviceLoginService() error = %v", err)
	}

	// Start a device login.
	resp, err := svc.Start(ctx, DeviceStartRequest{})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Approve it.
	err = svc.Approve(ctx, DeviceApproveRequest{
		DeviceCode: resp.DeviceCode,
		UserID:     "user-123",
	})
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	// Verify it's stored as approved.
	key := "device:" + resp.DeviceCode
	val, err := rc.Get(ctx, key).Result()
	if err != nil {
		t.Fatalf("redis get failed: %v", err)
	}
	parts := splitColon(val)
	if len(parts) < 3 || parts[1] != "approved" {
		t.Fatalf("stored value = %s, want approved status", val)
	}
}

func TestToken(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	defer rc.Close()

	svc, err := NewDeviceLoginService(rc)
	if err != nil {
		t.Fatalf("NewDeviceLoginService() error = %v", err)
	}

	// Start and approve.
	resp, err := svc.Start(ctx, DeviceStartRequest{})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	err = svc.Approve(ctx, DeviceApproveRequest{
		DeviceCode: resp.DeviceCode,
		UserID:     "user-123",
	})
	if err != nil {
		t.Fatalf("Approve() error = %v", err)
	}

	// Request token.
	tokenResp, err := svc.Token(ctx, DeviceTokenRequest{DeviceCode: resp.DeviceCode})
	if err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	if tokenResp.Token == "" {
		t.Fatal("Token() returned empty token")
	}
	if tokenResp.Status != "approved" {
		t.Fatalf("Status = %s, want approved", tokenResp.Status)
	}
}

func TestTokenNotApproved(t *testing.T) {
	rc := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rc.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis unavailable: %v", err)
	}
	defer rc.Close()

	svc, err := NewDeviceLoginService(rc)
	if err != nil {
		t.Fatalf("NewDeviceLoginService() error = %v", err)
	}

	// Start but don't approve.
	resp, err := svc.Start(ctx, DeviceStartRequest{})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Request token - should fail.
	_, err = svc.Token(ctx, DeviceTokenRequest{DeviceCode: resp.DeviceCode})
	if err == nil {
		t.Fatal("Token() should fail when not approved")
	}
	if err.Error() != "authorization_pending" {
		t.Fatalf("error = %v, want authorization_pending", err)
	}
}
