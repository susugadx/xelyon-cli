package history

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func newTestStorage(t *testing.T) *Storage {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage() error = %v", err)
	}
	return storage
}

func TestStorageSave_NilSession(t *testing.T) {
	storage := newTestStorage(t)
	if err := storage.Save(nil); err != nil {
		t.Fatalf("Save(nil) error = %v", err)
	}
}

func TestStorageRewrite_EmptySessionRemovesJSONL(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	session.Messages = nil
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite(empty) error = %v", err)
	}

	if _, err := os.Stat(storage.sessionPath(session.ID)); !os.IsNotExist(err) {
		t.Fatalf("session file should be removed, stat err = %v", err)
	}

	meta, err := storage.loadMetadata(session.ID)
	if err != nil {
		t.Fatalf("loadMetadata() error = %v", err)
	}
	if meta.MessageCount != 0 {
		t.Fatalf("MessageCount = %d, want 0", meta.MessageCount)
	}
	if meta.Preview != "" {
		t.Fatalf("Preview = %q, want empty", meta.Preview)
	}
}

func TestStorageLoad_SkipsCorruptedJSONLLine(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	f, err := os.OpenFile(storage.sessionPath(session.ID), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("os.OpenFile() error = %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString("{invalid json}\n"); err != nil {
		t.Fatalf("append invalid JSON error = %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("len(Messages) = %d, want 1", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "hello" {
		t.Fatalf("loaded content = %q, want %q", loaded.Messages[0].Content, "hello")
	}
}

func TestStorageLoad_MissingSessionFileAndInvalidMetadata(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if err := os.Remove(storage.sessionPath(session.ID)); err != nil {
		t.Fatalf("os.Remove() error = %v", err)
	}
	if _, err := storage.Load(session.ID); err == nil || !strings.Contains(err.Error(), "failed to open session file") {
		t.Fatalf("Load() error = %v, want open session file error", err)
	}

	if err := os.WriteFile(storage.metadataPath("broken"), []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("os.WriteFile(metadata) error = %v", err)
	}
	if _, err := storage.loadMetadata("broken"); err == nil || !strings.Contains(err.Error(), "failed to unmarshal metadata") {
		t.Fatalf("loadMetadata() error = %v, want unmarshal metadata error", err)
	}
}

func TestStorageSaveLoad_PersistsSessionResponseID(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("test-model")
	session.ResponseID = "resp_123"
	session.ResponseModel = "response-model"
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai"
	session.ResponseProviderName = "openai"
	session.ResponseProviderConfigKey = "openai"
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	meta, err := storage.loadMetadata(session.ID)
	if err != nil {
		t.Fatalf("loadMetadata() error = %v", err)
	}
	if meta.ResponseID != "resp_123" {
		t.Fatalf("metadata ResponseID = %q, want %q", meta.ResponseID, "resp_123")
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

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ResponseID != "resp_123" {
		t.Fatalf("loaded ResponseID = %q, want %q", loaded.ResponseID, "resp_123")
	}
	if loaded.ResponseModel != "response-model" {
		t.Fatalf("loaded ResponseModel = %q, want %q", loaded.ResponseModel, "response-model")
	}
	if loaded.ProviderName != "openai" || loaded.ProviderConfigKey != "openai" {
		t.Fatalf("loaded provider identity = (%q, %q), want (%q, %q)", loaded.ProviderName, loaded.ProviderConfigKey, "openai", "openai")
	}
	if loaded.ResponseProviderName != "openai" || loaded.ResponseProviderConfigKey != "openai" {
		t.Fatalf(
			"loaded response provider identity = (%q, %q), want (%q, %q)",
			loaded.ResponseProviderName,
			loaded.ResponseProviderConfigKey,
			"openai",
			"openai",
		)
	}
}

func TestStorageLoad_BackfillsLegacyResponseContextFromRuntimeIdentity(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("saved-model")
	session.ResponseID = "resp_legacy"
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai"
	session.AddMessage("user", "hello", "saved-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	metaPath := storage.metadataPath(session.ID)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(metadata) error = %v", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	meta.ResponseModel = ""
	meta.ResponseProviderName = ""
	meta.ResponseProviderConfigKey = ""
	meta.ResponseContextVersion = 0

	legacyRaw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(metadata) error = %v", err)
	}
	if err := os.WriteFile(metaPath, legacyRaw, 0o600); err != nil {
		t.Fatalf("WriteFile(metadata) error = %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ResponseModel != "saved-model" {
		t.Fatalf("loaded.ResponseModel = %q, want %q", loaded.ResponseModel, "saved-model")
	}
	if loaded.ResponseProviderName != "openai" || loaded.ResponseProviderConfigKey != "openai" {
		t.Fatalf(
			"loaded response provider identity = (%q, %q), want (%q, %q)",
			loaded.ResponseProviderName,
			loaded.ResponseProviderConfigKey,
			"openai",
			"openai",
		)
	}
}

func TestStorageLoad_VersionedResponseContextDoesNotGuessMissingProviderIdentity(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("saved-model")
	session.ResponseID = "resp_versioned"
	session.ResponseModel = "saved-model"
	session.ResponseProviderName = "openai"
	session.ResponseProviderConfigKey = "openai"
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai"
	session.AddMessage("user", "hello", "saved-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	metaPath := storage.metadataPath(session.ID)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(metadata) error = %v", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	meta.ResponseProviderName = ""
	meta.ResponseProviderConfigKey = ""
	versionedRaw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(metadata) error = %v", err)
	}
	if err := os.WriteFile(metaPath, versionedRaw, 0o600); err != nil {
		t.Fatalf("WriteFile(metadata) error = %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ResponseProviderName != "" || loaded.ResponseProviderConfigKey != "" {
		t.Fatalf(
			"loaded response provider identity = (%q, %q), want left empty for malformed versioned metadata",
			loaded.ResponseProviderName,
			loaded.ResponseProviderConfigKey,
		)
	}
}

func TestStorageLoad_LegacyResponseContextKeepsProviderOwnerEmptyWhenUnknown(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("saved-model")
	session.ResponseID = "resp_legacy"
	session.AddMessage("user", "hello", "saved-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	metaPath := storage.metadataPath(session.ID)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(metadata) error = %v", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	meta.ResponseModel = ""
	meta.ResponseProviderName = ""
	meta.ResponseProviderConfigKey = ""
	meta.ResponseContextVersion = 0

	legacyRaw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(metadata) error = %v", err)
	}
	if err := os.WriteFile(metaPath, legacyRaw, 0o600); err != nil {
		t.Fatalf("WriteFile(metadata) error = %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ResponseModel != "saved-model" {
		t.Fatalf("loaded.ResponseModel = %q, want %q", loaded.ResponseModel, "saved-model")
	}
	if loaded.ResponseProviderName != "openai" || loaded.ResponseProviderConfigKey != "" {
		t.Fatalf(
			"loaded response provider identity = (%q, %q), want (%q, %q)",
			loaded.ResponseProviderName,
			loaded.ResponseProviderConfigKey,
			"openai",
			"",
		)
	}
}

func TestStorageLoad_RepairsVersion1GuessedOpenAIOwnerForAliasRuntime(t *testing.T) {
	storage := newTestStorage(t)

	session := NewSession("saved-model")
	session.ResponseID = "resp_v1"
	session.ProviderName = "openai"
	session.ProviderConfigKey = "openai-alt"
	session.ResponseModel = "saved-model"
	session.ResponseProviderName = "openai"
	session.ResponseProviderConfigKey = "openai"
	session.AddMessage("user", "hello", "saved-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	metaPath := storage.metadataPath(session.ID)
	raw, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(metadata) error = %v", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	meta.ResponseContextVersion = 1
	version1Raw, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		t.Fatalf("MarshalIndent(metadata) error = %v", err)
	}
	if err := os.WriteFile(metaPath, version1Raw, 0o600); err != nil {
		t.Fatalf("WriteFile(metadata) error = %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.ResponseProviderName != "openai" || loaded.ResponseProviderConfigKey != "" {
		t.Fatalf(
			"loaded response provider identity = (%q, %q), want (%q, %q)",
			loaded.ResponseProviderName,
			loaded.ResponseProviderConfigKey,
			"openai",
			"",
		)
	}
}
