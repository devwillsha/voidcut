package auth

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestLoadFromMissingFile(t *testing.T) {
	_, err := LoadFrom(filepath.Join(t.TempDir(), "credentials"))
	if !errors.Is(err, ErrCredentialsNotFound) {
		t.Fatalf("LoadFrom() error = %v, want ErrCredentialsNotFound", err)
	}
}

func TestLoadFromRejectsMalformedAndExpiredCredentials(t *testing.T) {
	directory := t.TempDir()
	malformed := filepath.Join(directory, "malformed")
	if err := os.WriteFile(malformed, []byte("not-json"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFrom(malformed); err == nil {
		t.Fatal("LoadFrom() accepted malformed credentials")
	}

	expired := filepath.Join(directory, "expired")
	if err := SaveTo(expired, Credentials{Token: "token", ExpiresAt: time.Now().Add(-time.Minute)}); err == nil {
		t.Fatal("SaveTo() accepted expired credentials")
	}
}

func TestSaveAndLoadCredentials(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".voidcut", "credentials")
	credentials := Credentials{Token: "secret-token", UserID: "user-1", ExpiresAt: time.Now().Add(time.Hour)}
	if err := SaveTo(path, credentials); err != nil {
		t.Fatalf("SaveTo() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if permission := info.Mode().Perm(); permission != 0600 {
			t.Fatalf("credential permissions = %o, want 600", permission)
		}
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom() error = %v", err)
	}
	if loaded.Token != credentials.Token || loaded.UserID != credentials.UserID {
		t.Fatalf("loaded credentials = %+v, want %+v", loaded, credentials)
	}
}

func TestCredentialsValidate(t *testing.T) {
	if err := (Credentials{}).Validate(time.Now()); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("empty credentials error = %v", err)
	}
	if err := (Credentials{Token: "token", ExpiresAt: time.Now().Add(time.Minute)}).Validate(time.Now()); err != nil {
		t.Fatalf("valid credentials error = %v", err)
	}
}
