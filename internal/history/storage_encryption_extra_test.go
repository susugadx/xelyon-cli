package history

import (
	"bytes"
	"encoding/base64"
	"os"
	"strings"
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
	if bytes.Count(raw, []byte("\n")) != 1 {
		t.Fatalf("encrypted history should be stored as one JSONL line, got %d newlines", bytes.Count(raw, []byte("\n")))
	}

	decrypted, err := decryptStoredHistoryTestLine(raw, storage.passphrase)
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

	decrypted, err = decryptStoredHistoryTestLine(raw, storage.passphrase)
	if err != nil {
		t.Fatalf("DecryptSession() after rewrite error = %v", err)
	}
	if !bytes.Contains(decrypted, []byte("rewritten answer")) {
		t.Fatalf("decrypted rewritten payload = %q, want rewritten answer", decrypted)
	}
}

func TestStorage_RewriteCompactedState_WithEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XELYON_ENCRYPT_HISTORY", "1")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	session := NewSession("test-model")
	session.SetCompactedState([]CompactedItem{{
		Type:    "message",
		Role:    "user",
		Content: "compact secret prompt",
	}}, true)
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if bytes.Contains(raw, []byte("compact secret prompt")) {
		t.Fatal("encrypted compacted state should not contain plaintext content")
	}
	if bytes.Count(raw, []byte("\n")) != 1 {
		t.Fatalf("encrypted compacted state should be stored as one JSONL line, got %d newlines", bytes.Count(raw, []byte("\n")))
	}

	decrypted, err := decryptStoredHistoryTestLine(raw, storage.passphrase)
	if err != nil {
		t.Fatalf("DecryptSession() error = %v", err)
	}
	if !bytes.Contains(decrypted, []byte("compact secret prompt")) {
		t.Fatalf("decrypted payload = %q, want compacted prompt", decrypted)
	}
	if _, err := os.Stat(storage.sessionPath(session.ID)); err != nil {
		t.Fatalf("session file missing before Load: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !loaded.IsCompactedMode {
		t.Fatal("loaded.IsCompactedMode = false, want true")
	}
	if len(loaded.CompactedItems) != 1 || loaded.CompactedItems[0].Content != "compact secret prompt" {
		t.Fatalf("loaded.CompactedItems = %#v, want compact secret prompt", loaded.CompactedItems)
	}
}

func TestStorage_LoadLargeEncryptedCompactedState_WithBase64ExpandedLine(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XELYON_ENCRYPT_HISTORY", "1")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}

	largeData := strings.Repeat("x", 13*1024*1024)
	session := NewSession("test-model")
	session.SetCompactedState([]CompactedItem{{
		Type: "compacted",
		Data: largeData,
	}}, true)
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite() error = %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	line := bytes.TrimSuffix(raw, []byte("\n"))
	if len(line) <= maxSessionHistoryPlainLineBytes {
		t.Fatalf("encoded line length = %d, want over plaintext limit %d", len(line), maxSessionHistoryPlainLineBytes)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.CompactedItems) != 1 {
		t.Fatalf("len(loaded.CompactedItems) = %d, want 1", len(loaded.CompactedItems))
	}
	if loaded.CompactedItems[0].Data != largeData {
		t.Fatalf("loaded compacted data length = %d, want %d", len(loaded.CompactedItems[0].Data), len(largeData))
	}
}

func TestStorage_EncodeHistoryLineRejectsOversizedPlaintextLine(t *testing.T) {
	storage := &Storage{}
	msg := MessageEntry{
		Role:    "assistant",
		Content: strings.Repeat("x", maxSessionHistoryPlainLineBytes+1),
	}

	_, err := storage.encodeHistoryLine(msg)
	if err == nil {
		t.Fatal("encodeHistoryLine() error = nil, want plaintext size error")
	}
	if !strings.Contains(err.Error(), "plaintext line exceeds limit") {
		t.Fatalf("encodeHistoryLine() error = %v, want plaintext size detail", err)
	}
}

func TestStorage_EncodeHistoryLineRejectsOversizedEncryptedStoredLine(t *testing.T) {
	originalEncrypt := encryptSessionForStorage
	t.Cleanup(func() {
		encryptSessionForStorage = originalEncrypt
	})
	encryptSessionForStorage = func(_ []byte, _ string) ([]byte, error) {
		return bytes.Repeat([]byte("x"), base64.StdEncoding.DecodedLen(maxSessionHistoryStoredLineBytes)+1), nil
	}

	storage := &Storage{encryption: true, passphrase: "test"}
	_, err := storage.encodeHistoryLine(MessageEntry{Role: "user", Content: "hello"})
	if err == nil {
		t.Fatal("encodeHistoryLine() error = nil, want stored line size error")
	}
	if !strings.Contains(err.Error(), "stored line exceeds limit") {
		t.Fatalf("encodeHistoryLine() error = %v, want stored line size detail", err)
	}
}

func decryptStoredHistoryTestLine(raw []byte, passphrase string) ([]byte, error) {
	line := bytes.TrimSuffix(raw, []byte("\n"))
	ciphertext, ok := decodeEncryptedHistoryLine(line)
	if !ok {
		return nil, os.ErrInvalid
	}
	return crypto.DecryptSession(ciphertext, passphrase)
}
