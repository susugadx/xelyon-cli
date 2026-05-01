package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestStorage_SaveMetadata(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// セッション作成
	session := NewSession("test-model")
	session.AddMessage("user", "This is a long message that should be truncated in the preview because it exceeds the 80 character limit", "test-model")

	// 保存
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// メタデータファイルを読み込み
	metaPath := storage.metadataPath(session.ID)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("Failed to read metadata file: %v", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Failed to unmarshal metadata: %v", err)
	}

	// 検証
	if meta.ID != session.ID {
		t.Errorf("Expected ID %s, got %s", session.ID, meta.ID)
	}

	if meta.MessageCount != 1 {
		t.Errorf("Expected MessageCount 1, got %d", meta.MessageCount)
	}

	// プレビューが80文字で切り詰められているべき
	if len(meta.Preview) > 83 { // 80 + "..."
		t.Errorf("Expected preview to be truncated to ~80 chars, got %d chars", len(meta.Preview))
	}

	if !strings.HasSuffix(meta.Preview, "...") {
		t.Error("Expected preview to end with '...'")
	}
}

func TestStorage_SaveMetadata_MultibyteTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// 90 runes of Japanese text (each rune = 3 bytes, total = 270 bytes)
	// Byte-level truncation at 80 would split a multi-byte character.
	longJapanese := strings.Repeat("あ", 90) // 90 runes, 270 bytes
	session := NewSession("test-model")
	session.AddMessage("user", longJapanese, "test-model")

	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	metaPath := storage.metadataPath(session.ID)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("Failed to read metadata: %v", err)
	}

	var meta SessionMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("Failed to unmarshal metadata (possibly invalid UTF-8): %v", err)
	}

	if !utf8.ValidString(meta.Preview) {
		t.Error("Preview contains invalid UTF-8 after truncation")
	}

	if !strings.HasSuffix(meta.Preview, "...") {
		t.Error("Expected preview to end with '...'")
	}

	// Should be 80 runes + "..." (3 runes) = 83 runes total
	runeCount := utf8.RuneCountInString(meta.Preview)
	if runeCount != 83 {
		t.Errorf("Expected 83 runes (80 + '...'), got %d", runeCount)
	}
}

func TestStorage_SessionPath(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		baseDir: tmpDir,
	}

	sessionID := "test-session-123"
	path := storage.sessionPath(sessionID)

	expectedPath := filepath.Join(tmpDir, "test-session-123.jsonl")
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}
}

func TestStorage_MetadataPath(t *testing.T) {
	tmpDir := t.TempDir()
	storage := &Storage{
		baseDir: tmpDir,
	}

	sessionID := "test-session-456"
	path := storage.metadataPath(sessionID)

	expectedPath := filepath.Join(tmpDir, "metadata", "test-session-456.json")
	if path != expectedPath {
		t.Errorf("Expected path %s, got %s", expectedPath, path)
	}
}

func TestStorage_SaveMetadata_ReplacesExistingFileAtomically(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	session := NewSession("test-model")
	session.AddMessage("user", "first", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save(first) failed: %v", err)
	}

	session.ResetConversation()
	session.AddMessage("user", "second", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save(second) failed: %v", err)
	}

	metaPath := storage.metadataPath(session.ID)
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile(metadata) failed: %v", err)
	}
	var meta SessionMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("json.Unmarshal(metadata) failed: %v", err)
	}
	if meta.Preview != "second" {
		t.Fatalf("meta.Preview = %q, want %q", meta.Preview, "second")
	}
}
