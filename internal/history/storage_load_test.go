package history

import (
	"os"
	"strings"
	"testing"
)

func newStorageLoadTestStorage(t *testing.T) *Storage {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}
	return storage
}

func TestStorage_Save_LoadPendingApprovedPlan(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Hello", "test-model")
	session.PendingApprovedPlan = "Implementation Plan\n1. Restore me later"
	session.PendingApprovedPlanHasChanges = true
	session.PendingApprovedPlanChangedFiles = []string{"foo.go", "bar.go"}

	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.PendingApprovedPlan != session.PendingApprovedPlan {
		t.Fatalf("PendingApprovedPlan = %q, want %q", loaded.PendingApprovedPlan, session.PendingApprovedPlan)
	}
	if loaded.PendingApprovedPlanHasChanges != session.PendingApprovedPlanHasChanges {
		t.Fatalf("PendingApprovedPlanHasChanges = %v, want %v", loaded.PendingApprovedPlanHasChanges, session.PendingApprovedPlanHasChanges)
	}
	if strings.Join(loaded.PendingApprovedPlanChangedFiles, ",") != strings.Join(session.PendingApprovedPlanChangedFiles, ",") {
		t.Fatalf("PendingApprovedPlanChangedFiles = %v, want %v", loaded.PendingApprovedPlanChangedFiles, session.PendingApprovedPlanChangedFiles)
	}
}

func TestStorage_Load_MetadataOnlySession(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.PendingApprovedPlan = "Implementation Plan\n1. Restore me later"
	session.PendingApprovedPlanHasChanges = true
	session.PendingApprovedPlanChangedFiles = []string{"foo.go"}
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Messages) != 0 {
		t.Fatalf("len(loaded.Messages) = %d, want 0 for metadata-only session", len(loaded.Messages))
	}
	if loaded.PendingApprovedPlan != session.PendingApprovedPlan {
		t.Fatalf("PendingApprovedPlan = %q, want %q", loaded.PendingApprovedPlan, session.PendingApprovedPlan)
	}
	if loaded.PendingApprovedPlanHasChanges != session.PendingApprovedPlanHasChanges {
		t.Fatalf("PendingApprovedPlanHasChanges = %v, want %v", loaded.PendingApprovedPlanHasChanges, session.PendingApprovedPlanHasChanges)
	}
	if strings.Join(loaded.PendingApprovedPlanChangedFiles, ",") != strings.Join(session.PendingApprovedPlanChangedFiles, ",") {
		t.Fatalf("PendingApprovedPlanChangedFiles = %v, want %v", loaded.PendingApprovedPlanChangedFiles, session.PendingApprovedPlanChangedFiles)
	}
}

func TestStorage_Load_MissingHistoryFileForConversationSessionReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if err := os.Remove(storage.sessionPath(session.ID)); err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	_, err := storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want missing history file error")
	}
	if !strings.Contains(err.Error(), "metadata expects 1 messages") {
		t.Fatalf("Load() error = %v, want missing history detail", err)
	}
}

func TestStorage_Load_TruncatedConversationHistoryReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Hello", "test-model")
	session.AddMessage("assistant", "Hi there", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("len(lines) = %d, want 2", len(lines))
	}
	if err := os.WriteFile(storage.sessionPath(session.ID), []byte(lines[0]+"\n"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want inconsistency error")
	}
	if !strings.Contains(err.Error(), "metadata expects 2 messages, loaded 1") {
		t.Fatalf("Load() error = %v, want count mismatch detail", err)
	}
}

func TestStorage_Load_LegacyPlaintextHistoryAfterEncryptionToggle(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XELYON_ENCRYPT_HISTORY", "")

	plaintextStorage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "Plaintext message", "test-model")
	if err := plaintextStorage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	t.Setenv("XELYON_ENCRYPT_HISTORY", "1")
	encryptedStorage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage with encryption failed: %v", err)
	}

	loaded, err := encryptedStorage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed after encryption toggle: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("len(loaded.Messages) = %d, want 1", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Plaintext message" {
		t.Fatalf("loaded.Messages[0].Content = %q, want %q", loaded.Messages[0].Content, "Plaintext message")
	}
}

func TestStorage_Load_CorruptedData(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Valid message", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	filePath := storage.sessionPath(session.ID)
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	if _, err := f.WriteString("CORRUPTED LINE\n"); err != nil {
		t.Fatalf("Failed to write corrupted line: %v", err)
	}
	_ = f.Close()

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load should not fail on corrupted data: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Errorf("Expected 1 valid message (corrupted line skipped), got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Valid message" {
		t.Errorf("Expected 'Valid message', got '%s'", loaded.Messages[0].Content)
	}
}

func TestStorage_Load_CorruptedConversationRowWithToolAuditReturnsError(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	session.AddMessage("user", "Valid message", "test-model")
	session.AddMessage("assistant", "Assistant reply", "test-model")
	session.AddToolExecution("read_file", map[string]string{"path": "foo.go"}, "ok", true, "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	raw, err := os.ReadFile(storage.sessionPath(session.ID))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("len(lines) = %d, want 3", len(lines))
	}
	lines[1] = "CORRUPTED ASSISTANT LINE"
	if err := os.WriteFile(storage.sessionPath(session.ID), []byte(strings.Join(lines, "\n")+"\n"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	_, err = storage.Load(session.ID)
	if err == nil {
		t.Fatal("Load() error = nil, want inconsistency error")
	}
	if !strings.Contains(err.Error(), "metadata expects 2 messages, loaded 1") {
		t.Fatalf("Load() error = %v, want conversation count mismatch", err)
	}
}

func TestStorage_Load_LongConversationLine(t *testing.T) {
	storage := newStorageLoadTestStorage(t)

	session := NewSession("test-model")
	longContent := strings.Repeat("0123456789abcdef", 5*1024) // 80KiB
	session.AddMessage("assistant", longContent, "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed for long line: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("len(loaded.Messages) = %d, want 1", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != longContent {
		t.Fatal("loaded long message content mismatch")
	}
}
