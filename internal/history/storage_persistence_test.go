package history

import (
	"os"
	"strings"
	"testing"
)

func TestStorage_Save_Load(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "Hello", "test-model")

	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	session.AddMessage("assistant", "Hi there", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, loaded.ID)
	}
	if loaded.Model != session.Model {
		t.Errorf("Expected model %s, got %s", session.Model, loaded.Model)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(loaded.Messages))
	}
	if loaded.Messages[0].Content != "Hello" {
		t.Errorf("Expected first message 'Hello', got '%s'", loaded.Messages[0].Content)
	}
	if loaded.Messages[1].Content != "Hi there" {
		t.Errorf("Expected second message 'Hi there', got '%s'", loaded.Messages[1].Content)
	}
}

func TestStorage_Save_Append(t *testing.T) {
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
		t.Fatalf("First save failed: %v", err)
	}

	session.AddMessage("assistant", "Response 1", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("Expected 2 messages after append, got %d", len(loaded.Messages))
	}
}

func TestStorage_Save_DoesNotDuplicatePersistedMessages(t *testing.T) {
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
		t.Fatalf("First save failed: %v", err)
	}
	if err := storage.Save(session); err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Messages) != 1 {
		t.Fatalf("Expected 1 message after duplicate save, got %d", len(loaded.Messages))
	}
}

func TestStorage_Save_LoadToolExecutionEntry(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "hello", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save(user) failed: %v", err)
	}

	session.AddToolExecution("spawn_agent", map[string]string{"message": "read files"}, strings.Repeat("x", 250), true, "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save(tool execution) failed: %v", err)
	}

	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("Expected 2 stored entries, got %d", len(loaded.Messages))
	}
	entry := loaded.Messages[1]
	if entry.EntryType != toolExecutionEntryType {
		t.Fatalf("EntryType = %q, want %q", entry.EntryType, toolExecutionEntryType)
	}
	if entry.ToolExecution == nil {
		t.Fatal("ToolExecution = nil, want details")
	}
	if entry.ToolExecution.Name != "spawn_agent" {
		t.Fatalf("ToolExecution.Name = %q, want %q", entry.ToolExecution.Name, "spawn_agent")
	}
	if entry.ToolExecution.Args["message"] != "read files" {
		t.Fatalf("ToolExecution.Args[message] = %q, want %q", entry.ToolExecution.Args["message"], "read files")
	}
	if len([]rune(entry.ToolExecution.ResultPreview)) != 200 {
		t.Fatalf("len(ResultPreview) = %d, want 200", len([]rune(entry.ToolExecution.ResultPreview)))
	}

	sessions, err := storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("Expected 1 session metadata, got %d", len(sessions))
	}
	if sessions[0].MessageCount != 1 {
		t.Fatalf("MessageCount = %d, want 1 conversation message", sessions[0].MessageCount)
	}
}

func TestStorage_Save_EmptyMessages(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save with empty messages failed: %v", err)
	}

	filePath := storage.sessionPath(session.ID)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Expected no JSONL file to be created for empty messages")
	}
}
