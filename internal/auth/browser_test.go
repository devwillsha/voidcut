package auth

import "testing"

func TestValidateVerificationURL(t *testing.T) {
	valid := []string{
		"http://localhost:8080/connect?code=ABCD-1234",
		"https://voidcut.app/connect?code=ABCD-1234",
	}
	for _, rawURL := range valid {
		if err := ValidateVerificationURL(rawURL); err != nil {
			t.Errorf("ValidateVerificationURL(%q) error = %v", rawURL, err)
		}
	}

	invalid := []string{
		"",
		"javascript:alert(1)",
		"file:///tmp/credentials",
		"https://",
		"not a URL",
	}
	for _, rawURL := range invalid {
		if err := ValidateVerificationURL(rawURL); err == nil {
			t.Errorf("ValidateVerificationURL(%q) returned nil", rawURL)
		}
	}
}
