package history

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestStorage_Rewrite(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "Message 1", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	session.AddMessage("assistant", "Message 2", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("append save failed: %v", err)
	}

	session.Messages = []MessageEntry{{
		Timestamp: time.Now(),
		Role:      "user",
		Content:   "Cleared context summary",
		Model:     "test-model",
	}}
	if err := storage.Rewrite(session); err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded.Messages) != 1 {
		t.Fatalf("Expected 1 message after rewrite, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Cleared context summary" {
		t.Fatalf("loaded message content = %q, want %q", loaded.Messages[0].Content, "Cleared context summary")
	}
}

func TestStorage_RewriteLeavesExistingFileOnEncodeError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "kept history", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	filePath := storage.sessionPath(session.ID)
	before, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile before rewrite failed: %v", err)
	}

	session.ReplaceMessagesFromAPI([]api.Message{
		{Role: "user", Content: strings.Repeat("x", maxSessionHistoryPlainLineBytes)},
	}, "test-model")

	if err := storage.Rewrite(session); err == nil {
		t.Fatal("Rewrite() error = nil, want oversized line error")
	}

	after, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile after failed rewrite failed: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("history file changed after failed rewrite")
	}
	if !session.needsRewrite() {
		t.Fatal("session.needsRewrite() = false, want true after failed rewrite")
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load after failed rewrite failed: %v", err)
	}
	if len(loaded.Messages) != 1 || loaded.Messages[0].Content != "kept history" {
		t.Fatalf("loaded.Messages = %#v, want original history", loaded.Messages)
	}
}

func TestStorage_RewriteNilSession(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	if err := storage.Rewrite(nil); err != nil {
		t.Fatalf("Rewrite(nil) failed: %v", err)
	}
}

func TestStorage_SaveLeavesExistingFileOnEncodeErrorBeforeAppendingBatch(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "persisted history", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	filePath := storage.sessionPath(session.ID)
	before, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile before save failed: %v", err)
	}

	session.AddMessage("assistant", "small unsaved message", "test-model")
	session.AddMessage("user", strings.Repeat("x", maxSessionHistoryPlainLineBytes), "test-model")

	if err := storage.Save(session); err == nil {
		t.Fatal("Save() error = nil, want oversized line error")
	}

	after, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("ReadFile after failed save failed: %v", err)
	}
	if string(after) != string(before) {
		t.Fatal("history file changed after failed save")
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load after failed save failed: %v", err)
	}
	if len(loaded.Messages) != 1 || loaded.Messages[0].Content != "persisted history" {
		t.Fatalf("loaded.Messages = %#v, want original history", loaded.Messages)
	}
}

func TestStorage_SaveRewritesAfterReplaceMessagesFromAPI(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "old question", "test-model")
	session.AddMessage("assistant", "old answer", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("initial save failed: %v", err)
	}

	session.ReplaceMessagesFromAPI([]api.Message{
		{Role: "system", Content: "compressed summary"},
	}, "test-model")
	if !session.needsRewrite() {
		t.Fatal("session.needsRewrite() = false, want true after message replacement")
	}

	badStorage := *storage
	badStorage.baseDir = filepath.Join(tmpDir, "missing-history-dir")
	if err := badStorage.Rewrite(session); err == nil {
		t.Fatal("Rewrite with missing history dir error = nil, want error")
	}
	if !session.needsRewrite() {
		t.Fatal("session.needsRewrite() = false, want true after failed rewrite")
	}

	if err := storage.Save(session); err != nil {
		t.Fatalf("Save after replacement failed: %v", err)
	}
	if session.needsRewrite() {
		t.Fatal("session.needsRewrite() = true, want false after successful rewrite save")
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("len(loaded.Messages) = %d, want 1", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "compressed summary" {
		t.Fatalf("loaded message content = %q, want compressed summary", loaded.Messages[0].Content)
	}
}
