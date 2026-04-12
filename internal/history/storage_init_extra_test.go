package history

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withStorageInitHooks(t *testing.T) {
	t.Helper()

	oldUserHomeDir := userHomeDirForStorage
	oldGetPassphrase := getPassphraseForStorage
	t.Cleanup(func() {
		userHomeDirForStorage = oldUserHomeDir
		getPassphraseForStorage = oldGetPassphrase
	})
}

func TestNewStorage_ErrorPaths(t *testing.T) {
	t.Run("returns home dir lookup error", func(t *testing.T) {
		withStorageInitHooks(t)
		userHomeDirForStorage = func() (string, error) {
			return "", errors.New("home lookup failed")
		}

		if _, err := NewStorage(); err == nil || !strings.Contains(err.Error(), "failed to get home dir") {
			t.Fatalf("NewStorage() error = %v, want home dir error", err)
		}
	})

	t.Run("returns history dir creation error", func(t *testing.T) {
		homeFile := filepath.Join(t.TempDir(), "home-file")
		if err := os.WriteFile(homeFile, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		t.Setenv("HOME", homeFile)

		if _, err := NewStorage(); err == nil || !strings.Contains(err.Error(), "failed to create history dir") {
			t.Fatalf("NewStorage() error = %v, want create history dir error", err)
		}
	})

	t.Run("returns encryption key error", func(t *testing.T) {
		withStorageInitHooks(t)
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XELYON_ENCRYPT_HISTORY", "1")
		getPassphraseForStorage = func() (string, error) {
			return "", errors.New("key generation failed")
		}

		if _, err := NewStorage(); err == nil || !strings.Contains(err.Error(), "failed to get encryption key") {
			t.Fatalf("NewStorage() error = %v, want encryption key error", err)
		}
	})
}
