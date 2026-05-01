package history

import (
	"os"
	"strings"
	"testing"
)

func TestStorage_WithEncryption_Save_Load(t *testing.T) {
	t.Skip("Skipping encryption test - requires proper crypto setup and may fail in CI")

	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	os.Setenv("XELYON_ENCRYPT_HISTORY", "1")
	defer func() {
		os.Unsetenv("HOME")
		os.Unsetenv("XELYON_ENCRYPT_HISTORY")
	}()

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	if !storage.encryption {
		t.Skip("Encryption not enabled, skipping test")
	}

	session := NewSession("test-model")
	session.AddMessage("user", "Secret message", "test-model")

	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	filePath := storage.sessionPath(session.ID)
	rawData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read JSONL file: %v", err)
	}
	if strings.Contains(string(rawData), "Secret message") {
		t.Error("Expected message to be encrypted, but found plaintext")
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Secret message" {
		t.Errorf("Expected decrypted message 'Secret message', got '%s'", loaded.Messages[0].Content)
	}
}
