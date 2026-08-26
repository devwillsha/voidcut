package auth

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// ValidateVerificationURL accepts only browser-safe HTTP(S) URLs returned by
// the Gateway device-login endpoint.
func ValidateVerificationURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid verification URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("verification URL must use HTTP or HTTPS")
	}
	return nil
}

// OpenBrowser launches the default browser without invoking a shell.
func OpenBrowser(ctx context.Context, rawURL string) error {
	if err := ValidateVerificationURL(rawURL); err != nil {
		return err
	}

	var command string
	var args []string
	switch runtime.GOOS {
	case "windows":
		command = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", rawURL}
	case "darwin":
		command = "open"
		args = []string{rawURL}
	default:
		command = "xdg-open"
		args = []string{rawURL}
	}

	if err := exec.CommandContext(ctx, command, args...).Run(); err != nil {
		return fmt.Errorf("open verification URL: %w", err)
	}
	return nil
}
