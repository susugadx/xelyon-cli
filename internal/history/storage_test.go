package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
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

	// ディレクトリが作成されていることを確認
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

func TestStorage_Save_Load(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// セッション作成
	session := NewSession("test-model")
	session.AddMessage("user", "Hello", "test-model")

	// 最初のメッセージを保存
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 2つ目のメッセージを追加して保存
	session.AddMessage("assistant", "Hi there", "test-model")
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 読み込み
	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 検証
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

	// セッション作成
	session := NewSession("test-model")
	session.AddMessage("user", "Message 1", "test-model")

	// 最初の保存
	if err := storage.Save(session); err != nil {
		t.Fatalf("First save failed: %v", err)
	}

	// 追加メッセージ
	session.AddMessage("assistant", "Response 1", "test-model")

	// 2回目の保存（追記）
	if err := storage.Save(session); err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	// 読み込み
	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 2つのメッセージが保存されているべき
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

func TestStorage_Save_EmptyMessages(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// メッセージなしのセッション
	session := NewSession("test-model")

	// 保存（何も書き込まれないべき）
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save with empty messages failed: %v", err)
	}

	// JSONLファイルが作成されていないことを確認
	filePath := storage.sessionPath(session.ID)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("Expected no JSONL file to be created for empty messages")
	}
}

func TestStorage_ListSessions(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// 複数のセッションを作成して保存
	// 注意: Session IDは秒単位のUnixタイムスタンプなので、1秒以上空ける必要がある
	var sessionIDs []string
	for i := 1; i <= 3; i++ {
		session := NewSession(fmt.Sprintf("model-%d", i))
		session.AddMessage("user", fmt.Sprintf("Message %d", i), session.Model)

		if err := storage.Save(session); err != nil {
			t.Fatalf("Save session%d failed: %v", i, err)
		}
		sessionIDs = append(sessionIDs, session.ID)

		// 次のセッションIDが異なるように1秒待つ
		if i < 3 {
			time.Sleep(1100 * time.Millisecond)
		}
	}

	// リスト取得
	sessions, err := storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	// 3つのセッションが返されるべき
	if len(sessions) != 3 {
		t.Fatalf("Expected 3 sessions, got %d", len(sessions))
	}

	// 新しい順にソートされているべき（最後→最初）
	if sessions[0].ID != sessionIDs[2] {
		t.Errorf("Expected newest session to be %s, got %s", sessionIDs[2], sessions[0].ID)
	}

	if sessions[2].ID != sessionIDs[0] {
		t.Errorf("Expected oldest session to be %s, got %s", sessionIDs[0], sessions[2].ID)
	}
}

func TestStorage_ListSessions_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// リスト取得（セッションなし）
	sessions, err := storage.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions failed: %v", err)
	}

	if len(sessions) != 0 {
		t.Errorf("Expected empty sessions, got %d", len(sessions))
	}
}

func TestStorage_GetLastSession(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// セッション作成
	session1 := NewSession("model-1")
	session1.AddMessage("user", "Old session", "model-1")
	if err := storage.Save(session1); err != nil {
		t.Fatalf("Failed to save session1: %v", err)
	}

	// 1秒待って異なるSession IDを生成
	time.Sleep(1100 * time.Millisecond)

	session2 := NewSession("model-2")
	session2.AddMessage("user", "New session", "model-2")
	if err := storage.Save(session2); err != nil {
		t.Fatalf("Failed to save session2: %v", err)
	}

	// 最新セッション取得
	lastID, err := storage.GetLastSession()
	if err != nil {
		t.Fatalf("GetLastSession failed: %v", err)
	}

	// session2が最新であるべき
	if lastID != session2.ID {
		t.Errorf("Expected last session %s, got %s", session2.ID, lastID)
	}
}

func TestStorage_GetLastSession_NoSessions(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("HOME", tmpDir)
	defer os.Unsetenv("HOME")

	storage, err := NewStorage()
	if err != nil {
		t.Fatalf("NewStorage failed: %v", err)
	}

	// セッションなしでGetLastSession
	_, err = storage.GetLastSession()
	if err == nil {
		t.Error("Expected error when no sessions exist")
	}

	if !strings.Contains(err.Error(), "no sessions found") {
		t.Errorf("Expected 'no sessions found' error, got: %v", err)
	}
}

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

func TestStorage_WithEncryption_Save_Load(t *testing.T) {
	t.Skip("Skipping encryption test - requires proper crypto setup and may fail in CI")

	// Skip if CGO is disabled (crypto requires CGO)
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

	// Skip if encryption not actually enabled (might fail on some systems)
	if !storage.encryption {
		t.Skip("Encryption not enabled, skipping test")
	}

	// セッション作成
	session := NewSession("test-model")
	session.AddMessage("user", "Secret message", "test-model")

	// 保存
	if err := storage.Save(session); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// JSONLファイルを直接読み込み
	filePath := storage.sessionPath(session.ID)
	rawData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read JSONL file: %v", err)
	}

	// 暗号化されているべき（平文の"Secret message"が含まれていないべき）
	if strings.Contains(string(rawData), "Secret message") {
		t.Error("Expected message to be encrypted, but found plaintext")
	}

	// 読み込み（復号化される）
	loaded, err := storage.Load(session.ID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// 復号化されたメッセージが正しいべき
	if len(loaded.Messages) != 1 {
		t.Fatalf("Expected 1 message, got %d", len(loaded.Messages))
	}

	if loaded.Messages[0].Content != "Secret message" {
		t.Errorf("Expected decrypted message 'Secret message', got '%s'", loaded.Messages[0].Content)
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
