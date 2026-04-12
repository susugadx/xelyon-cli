package history

import (
	"bytes"
	"os"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/crypto"
)

func TestStorage_SaveLoadRewrite_WithEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XELYON_ENCRYPT_HISTORY", "1")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "top secret prompt", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if bytes.Contains(raw, []byte("top secret prompt")) {
		t.Fatal("encrypted history file should not contain plaintext content")
	}

	decrypted, err := crypto.DecryptSession(bytes.TrimSuffix(raw, []byte("\n")), storage.passphrase)
	if err != nil {
		t.Fatalf("DecryptSession() error = %v", err)
	}
	if !bytes.Contains(decrypted, []byte("top secret prompt")) {
		t.Fatalf("decrypted payload = %q, want original prompt", decrypted)
	}

	session.Messages = []MessageEntry{{
		Timestamp: session.StartTime,
		Role:      "assistant",
		Content:   "rewritten answer",
		Model:     "test-model",
	}}
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	raw, err = os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile() after rewrite error = %v", err)
	}
	if bytes.Contains(raw, []byte("rewritten answer")) {
		t.Fatal("rewritten encrypted history file should not contain plaintext content")
	}

	decrypted, err = crypto.DecryptSession(bytes.TrimSuffix(raw, []byte("\n")), storage.passphrase)
	if err != nil {
		t.Fatalf("DecryptSession() after rewrite error = %v", err)
	}
	if !bytes.Contains(decrypted, []byte("rewritten answer")) {
		t.Fatalf("decrypted rewritten payload = %q, want rewritten answer", decrypted)
	}
}
