package agent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewID creates a random identifier for a recording session or trace.
func NewID(prefix string) (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate %s ID: %w", prefix, err)
	}
	return prefix + "-" + hex.EncodeToString(bytes), nil
}
