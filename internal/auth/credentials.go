// Package auth manages credentials used by the local Voidcut agent.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const credentialDirectory = ".voidcut"
const credentialFile = "credentials"

var ErrCredentialsNotFound = errors.New("credentials not found")
var ErrInvalidCredentials = errors.New("invalid credentials")

// Credentials contains the long-lived token used by agent traffic.
type Credentials struct {
	Token     string    `json:"token"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

// Path returns the default credentials path for the current user.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find user home directory: %w", err)
	}
	return filepath.Join(home, credentialDirectory, credentialFile), nil
}

// Load reads and validates credentials from the default credentials path.
func Load() (Credentials, error) {
	path, err := Path()
	if err != nil {
		return Credentials{}, err
	}
	return LoadFrom(path)
}

// LoadFrom reads and validates credentials from path.
func LoadFrom(path string) (Credentials, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Credentials{}, ErrCredentialsNotFound
		}
		return Credentials{}, fmt.Errorf("read credentials: %w", err)
	}

	var credentials Credentials
	if err := json.Unmarshal(raw, &credentials); err != nil {
		return Credentials{}, fmt.Errorf("decode credentials: %w", err)
	}
	if err := credentials.Validate(time.Now()); err != nil {
		return Credentials{}, err
	}
	return credentials, nil
}

// Save writes credentials with owner-only permissions.
func Save(credentials Credentials) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return SaveTo(path, credentials)
}

// SaveTo writes credentials to path with owner-only permissions.
func SaveTo(path string, credentials Credentials) error {
	if err := credentials.Validate(time.Now()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create credential directory: %w", err)
	}

	raw, err := json.MarshalIndent(credentials, "", "  ")
	if err != nil {
		return fmt.Errorf("encode credentials: %w", err)
	}
	raw = append(raw, '\n')

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open credentials: %w", err)
	}
	defer file.Close()
	if err := file.Chmod(0600); err != nil {
		return fmt.Errorf("set credential permissions: %w", err)
	}
	if _, err := file.Write(raw); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

// Validate checks the fields needed for authenticated agent traffic.
func (credentials Credentials) Validate(now time.Time) error {
	if strings.TrimSpace(credentials.Token) == "" {
		return ErrInvalidCredentials
	}
	if !credentials.ExpiresAt.IsZero() && !credentials.ExpiresAt.After(now) {
		return ErrInvalidCredentials
	}
	return nil
}
