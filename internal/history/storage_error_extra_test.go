package history

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withStorageCryptoHooks(t *testing.T) {
	t.Helper()

	oldEncrypt := encryptSessionForStorage
	oldDecrypt := decryptSessionForStorage
	t.Cleanup(func() {
		encryptSessionForStorage = oldEncrypt
		decryptSessionForStorage = oldDecrypt
	})
}

func TestStorageSaveAndRewrite_ErrorPermutations(t *testing.T) {
	t.Run("save reports encryption dependency error after file open", func(t *testing.T) {
		withStorageCryptoHooks(t)
		encryptSessionForStorage = func([]byte, string) ([]byte, error) {
			return nil, errors.New("rand failed")
		}

		storage := newTestStorage(t)
		storage.encryption = true
		storage.passphrase = "test-pass"
		session := NewSession("test-model")
		session.AddMessage("user", "hello", "test-model")

		if err := storage.Save(session); err == nil || !strings.Contains(err.Error(), "failed to encrypt message") {
			t.Fatalf("Save() error = %v, want encrypt message error", err)
		}
	})

	t.Run("save with unsaved messages reports open history file error", func(t *testing.T) {
		baseDir := t.TempDir()
		blocker := filepath.Join(baseDir, "blocked")
		if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		storage := &Storage{baseDir: blocker}
		session := NewSession("test-model")
		session.AddMessage("user", "hello", "test-model")

		if err := storage.Save(session); err == nil || !strings.Contains(err.Error(), "failed to open history file") {
			t.Fatalf("Save() error = %v, want open history file error", err)
		}
	})

	t.Run("save without unsaved messages reports metadata error", func(t *testing.T) {
		baseDir := t.TempDir()
		blocker := filepath.Join(baseDir, "blocked")
		if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		storage := &Storage{baseDir: blocker}
		session := NewSession("test-model")
		session.markPersisted()

		if err := storage.Save(session); err == nil || !strings.Contains(err.Error(), "failed to create metadata dir") {
			t.Fatalf("Save() error = %v, want metadata dir error", err)
		}
	})

	t.Run("rewrite with messages reports rewrite history file error", func(t *testing.T) {
		baseDir := t.TempDir()
		blocker := filepath.Join(baseDir, "blocked")
		if err := os.WriteFile(blocker, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}

		storage := &Storage{baseDir: blocker}
		session := NewSession("test-model")
		session.AddMessage("user", "hello", "test-model")

		if err := storage.Rewrite(session); err == nil || !strings.Contains(err.Error(), "failed to rewrite history file") {
			t.Fatalf("Rewrite() error = %v, want rewrite history file error", err)
		}
	})

	t.Run("rewrite reports encryption dependency error after file open", func(t *testing.T) {
		withStorageCryptoHooks(t)
		encryptSessionForStorage = func([]byte, string) ([]byte, error) {
			return nil, errors.New("cipher init failed")
		}

		storage := newTestStorage(t)
		storage.encryption = true
		storage.passphrase = "test-pass"
		session := NewSession("test-model")
		session.AddMessage("user", "hello", "test-model")

		if err := storage.Rewrite(session); err == nil || !strings.Contains(err.Error(), "failed to encrypt message") {
			t.Fatalf("Rewrite() error = %v, want encrypt message error", err)
		}
	})

	t.Run("rewrite empty session reports remove error when session path is non-empty dir", func(t *testing.T) {
		storage := newTestStorage(t)
		session := NewSession("test-model")
		sessionPath := storage.sessionPath(session.ID)
		if err := os.MkdirAll(sessionPath, 0o755); err != nil {
			t.Fatalf("MkdirAll(sessionPath) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(sessionPath, "child"), []byte("x"), 0o600); err != nil {
			t.Fatalf("WriteFile(child) error = %v", err)
		}

		if err := storage.Rewrite(session); err == nil || !strings.Contains(err.Error(), "failed to remove history file") {
			t.Fatalf("Rewrite(empty) error = %v, want remove history file error", err)
		}
	})
}

func TestStorageLoad_LongLineSucceedsWithinConfiguredBuffer(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("test-model")
	longContent := strings.Repeat("x", 70*1024)
	session.AddMessage("user", longContent, "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v, want success for long valid line", err)
	}
	if len(loaded.Messages) != 1 || loaded.Messages[0].Content != longContent {
		t.Fatalf("loaded.Messages = %#v, want single long message", loaded.Messages)
	}
}

func TestStorageLoad_EncryptedDecryptionFailureSkipsEntry(t *testing.T) {
	withStorageCryptoHooks(t)
	decryptSessionForStorage = func([]byte, string) ([]byte, error) {
		return nil, errors.New("decryption failed")
	}

	storage := newTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if err := os.WriteFile(storage.sessionPath(session.ID), raw, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	storage.encryption = true
	storage.passphrase = "wrong-passphrase"
	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v, want decrypt failure to be skipped", err)
	}
	if len(loaded.Messages) != 0 {
		t.Fatalf("len(Messages) = %d, want 0 when encrypted rows fail to decrypt", len(loaded.Messages))
	}
}

func TestStorageLoad_EncryptedRecordPreservesTrailingCarriageReturnByte(t *testing.T) {
	withStorageCryptoHooks(t)

	encryptSessionForStorage = func(data []byte, _ string) ([]byte, error) {
		encrypted := append([]byte(nil), data...)
		return append(encrypted, '\r'), nil
	}
	decryptSessionForStorage = func(data []byte, _ string) ([]byte, error) {
		if len(data) == 0 || data[len(data)-1] != '\r' {
			return nil, errors.New("trailing carriage return byte was lost")
		}
		return data[:len(data)-1], nil
	}

	storage := newTestStorage(t)
	storage.encryption = true
	storage.passphrase = "test-pass"

	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Messages) != 1 || loaded.Messages[0].Content != "hello" {
		t.Fatalf("loaded.Messages = %#v, want encrypted record to survive trailing carriage return byte", loaded.Messages)
	}
}

func TestStorageListSessionsAndGetLastSession_ErrorPaths(t *testing.T) {
	t.Run("list sessions skips broken metadata and non-json files", func(t *testing.T) {
		storage := newTestStorage(t)

		session := NewSession("test-model")
		session.AddMessage("user", "hello", "test-model")
		if err := storage.Save(session); err != nil {
			t.Fatalf("Save() error = %v", err)
		}

		metaDir := filepath.Join(storage.baseDir, "metadata")
		if err := os.WriteFile(filepath.Join(metaDir, "broken.json"), []byte("{invalid"), 0o600); err != nil {
			t.Fatalf("WriteFile(broken metadata) error = %v", err)
		}
		if err := os.WriteFile(filepath.Join(metaDir, "note.txt"), []byte("ignore"), 0o600); err != nil {
			t.Fatalf("WriteFile(non-json) error = %v", err)
		}

		sessions, err := storage.ListSessions()
		if err != nil {
			t.Fatalf("ListSessions() error = %v", err)
		}
		if len(sessions) != 1 || sessions[0].ID != session.ID {
			t.Fatalf("ListSessions() = %#v, want only valid saved session", sessions)
		}
	})

	t.Run("list sessions propagates readdir error", func(t *testing.T) {
		baseDir := t.TempDir()
		metaDir := filepath.Join(baseDir, "metadata")
		if err := os.WriteFile(metaDir, []byte("not a directory"), 0o600); err != nil {
			t.Fatalf("WriteFile(metadata blocker) error = %v", err)
		}

		storage := &Storage{baseDir: baseDir}
		if _, err := storage.ListSessions(); err == nil || !strings.Contains(err.Error(), "failed to read metadata dir") {
			t.Fatalf("ListSessions() error = %v, want read metadata dir error", err)
		}
		if _, err := storage.GetLastSession(); err == nil || !strings.Contains(err.Error(), "failed to read metadata dir") {
			t.Fatalf("GetLastSession() error = %v, want propagated read metadata dir error", err)
		}
	})
}

func TestStorageMetadata_PersistsResponseIDInFile(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("test-model")
	session.ResponseID = "resp_meta"
	session.ResponseModel = "response-model"
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai"
	session.ResponseProviderName = "openai"
	session.ResponseProviderConfigKey = "openai"
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	raw, err := os.ReadFile(storage.metadataPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile(metadata) error = %v", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	if meta.ResponseID != "resp_meta" {
		t.Fatalf("metadata ResponseID = %q, want %q", meta.ResponseID, "resp_meta")
	}
	if meta.ResponseContextVersion != responseContextMetadataVersion {
		t.Fatalf("metadata ResponseContextVersion = %d, want %d", meta.ResponseContextVersion, responseContextMetadataVersion)
	}
	if meta.ResponseModel != "response-model" {
		t.Fatalf("metadata ResponseModel = %q, want %q", meta.ResponseModel, "response-model")
	}
	if meta.ProviderName != "openai" || meta.ProviderConfigKey != "openai" {
		t.Fatalf("metadata provider identity = (%q, %q), want (%q, %q)", meta.ProviderName, meta.ProviderConfigKey, "openai", "openai")
	}
	if meta.ResponseProviderName != "openai" || meta.ResponseProviderConfigKey != "openai" {
		t.Fatalf(
			"metadata response provider identity = (%q, %q), want (%q, %q)",
			meta.ResponseProviderName,
			meta.ResponseProviderConfigKey,
			"openai",
			"openai",
		)
	}
}
