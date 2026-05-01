package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStorage(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	if storage == nil {
		t.Fatal("NewStorage returned nil")
	}

	expectedDir := filepath.Join(tmpDir, ".xelyon/history")
	if storage.baseDir != expectedDir {
		t.Errorf("Expected baseDir %s, got %s", expectedDir, storage.baseDir)
	}
	if _, err := os.Stat(expectedDir); os.IsNotExist(err) {
		t.Error("History directory was not created")
	}
}

func TestNewStorage_WithEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Setenv("XELYON_ENCRYPT_HISTORY", "1")
	defer func() {
		os.Unsetenv("HOME")
		os.Unsetenv("XELYON_ENCRYPT_HISTORY")
	}()

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage with encryption failed: %v", err)
	}
	if !storage.encryption {
		t.Error("Expected encryption to be enabled")
	}
	if storage.passphrase == "" {
		t.Error("Expected non-empty passphrase")
	}
}
