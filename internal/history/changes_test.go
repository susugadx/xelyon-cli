package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestNewChangeStorage(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	if cs == nil {
		t.Fatal("NewChangeStorage() returned nil")
	}

	// changesディレクトリが作成されたことを確認
	home, _ := os.UserHomeDir()
	changesPath := filepath.Join(home, changesDir)
	if _, err := os.Stat(changesPath); os.IsNotExist(err) {
		t.Error("NewChangeStorage() did not create changes directory")
	}
}

func TestAppendChange_Success(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	sessionID := "test-session-123"
	change := ChangeRecordInput{
		FilePath:    "/tmp/test.txt",
		Timestamp:   time.Now(),
		Tool:        "write_file",
		Description: "Test change",
	}

	err = cs.AppendChange(sessionID, change)
	if err != nil {
		t.Errorf("AppendChange() error = %v", err)
	}

	// ファイルが作成されたことを確認
	filename := filepath.Join(cs.changesPath, changeFileName(sessionID))
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("AppendChange() did not create changes file")
	}
}

func TestAppendChange_Multiple(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	sessionID := "test-session-multi"

	// 複数の変更を追加
	for i := 0; i < 3; i++ {
		change := ChangeRecordInput{
			FilePath:    "/tmp/test.txt",
			Timestamp:   time.Now(),
			Tool:        "write_file",
			Description: "Test change",
		}
		if err := cs.AppendChange(sessionID, change); err != nil {
			t.Errorf("AppendChange() error = %v", err)
		}
	}

	// 読み込んで数を確認
	changes, err := cs.LoadSessionChanges(sessionID)
	if err != nil {
		t.Fatalf("LoadSessionChanges() error = %v", err)
	}

	if len(changes) != 3 {
		t.Errorf("LoadSessionChanges() loaded %d changes, want 3", len(changes))
	}
}

func TestLoadSessionChanges_NoFile(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	sessionID := "nonexistent-session"

	changes, err := cs.LoadSessionChanges(sessionID)
	if err != nil {
		t.Errorf("LoadSessionChanges() error = %v", err)
	}

	if len(changes) != 0 {
		t.Errorf("LoadSessionChanges() returned %d changes, want 0 for nonexistent session", len(changes))
	}
}

func TestLoadSessionChanges_Success(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	sessionID := "test-session-load"
	change := ChangeRecordInput{
		FilePath:    "/tmp/test.txt",
		Timestamp:   time.Now(),
		Tool:        "write_file",
		Description: "Test change",
	}

	// 追加
	if err := cs.AppendChange(sessionID, change); err != nil {
		t.Fatalf("AppendChange() error = %v", err)
	}

	// 読み込み
	changes, err := cs.LoadSessionChanges(sessionID)
	if err != nil {
		t.Errorf("LoadSessionChanges() error = %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("LoadSessionChanges() loaded %d changes, want 1", len(changes))
	}

	// 内容確認
	loaded := changes[0]
	if loaded.SessionID != sessionID {
		t.Errorf("LoadSessionChanges() SessionID = %v, want %v", loaded.SessionID, sessionID)
	}
	if loaded.Tool != "write_file" {
		t.Errorf("LoadSessionChanges() Tool = %v, want 'write_file'", loaded.Tool)
	}
	if loaded.FilePath != "/tmp/test.txt" {
		t.Errorf("LoadSessionChanges() FilePath = %v, want '/tmp/test.txt'", loaded.FilePath)
	}
}

func TestLoadSessionChanges_CorruptedLine(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	sessionID := "test-session-corrupt"

	// 手動で破損したJSONLファイルを作成
	filename := filepath.Join(cs.changesPath, changeFileName(sessionID))
	content := `{"session_id":"test-session-corrupt","timestamp":"2024-01-01T00:00:00Z","tool":"write_file","file_path":"/tmp/test.txt","description":"Valid line"}
{invalid json line}
{"session_id":"test-session-corrupt","timestamp":"2024-01-01T00:00:01Z","tool":"str_replace","file_path":"/tmp/test2.txt","description":"Another valid line"}
`
	if err := os.WriteFile(filename, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 読み込み（破損行はスキップされる）
	changes, err := cs.LoadSessionChanges(sessionID)
	if err != nil {
		t.Errorf("LoadSessionChanges() error = %v", err)
	}

	// 破損行を除いた2行が読み込まれる
	if len(changes) != 2 {
		t.Errorf("LoadSessionChanges() loaded %d changes, want 2 (skipping corrupted line)", len(changes))
	}
}

func TestListSessions_Multiple(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	// 複数のセッションを作成
	sessions := []string{"session-1", "session-2", "session-3"}
	for _, sessionID := range sessions {
		change := ChangeRecordInput{
			FilePath:    "/tmp/test.txt",
			Timestamp:   time.Now(),
			Tool:        "write_file",
			Description: "Test change",
		}
		if err := cs.AppendChange(sessionID, change); err != nil {
			t.Fatalf("AppendChange() error = %v", err)
		}
		time.Sleep(10 * time.Millisecond) // タイムスタンプを確実に異なるものにする
	}

	// セッション一覧取得
	sessionInfos, err := cs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	if len(sessionInfos) < 3 {
		t.Errorf("ListSessions() returned %d sessions, want at least 3", len(sessionInfos))
	}

	// 最新順にソートされていることを確認
	for i := 1; i < len(sessionInfos); i++ {
		if sessionInfos[i].LastChange.After(sessionInfos[i-1].LastChange) {
			t.Error("ListSessions() sessions are not sorted by LastChange (newest first)")
		}
	}
}

func TestListSessions_EmptyDirectory(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	sessionInfos, err := cs.ListSessions()
	if err != nil {
		t.Errorf("ListSessions() error = %v", err)
	}

	if len(sessionInfos) != 0 {
		t.Errorf("ListSessions() returned %d sessions, want 0 for empty directory", len(sessionInfos))
	}
}

func TestCleanupOldChanges_Default(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	// 古いファイルを作成（31日前）
	oldSessionID := "old-session"
	oldFilename := filepath.Join(cs.changesPath, changeFileName(oldSessionID))
	if err := os.WriteFile(oldFilename, []byte("{}\n"), 0600); err != nil {
		t.Fatalf("Failed to create old file: %v", err)
	}

	// タイムスタンプを31日前に設定
	oldTime := time.Now().AddDate(0, 0, -31)
	if err := os.Chtimes(oldFilename, oldTime, oldTime); err != nil {
		t.Fatalf("Failed to change file time: %v", err)
	}

	// 新しいファイルを作成
	newSessionID := "new-session"
	change := ChangeRecordInput{
		FilePath:    "/tmp/test.txt",
		Timestamp:   time.Now(),
		Tool:        "write_file",
		Description: "Test change",
	}
	if err := cs.AppendChange(newSessionID, change); err != nil {
		t.Fatalf("AppendChange() error = %v", err)
	}

	// クリーンアップ（デフォルト30日）
	deletedCount, err := cs.CleanupOldChanges(30)
	if err != nil {
		t.Fatalf("CleanupOldChanges() error = %v", err)
	}

	if deletedCount != 1 {
		t.Errorf("CleanupOldChanges() deleted %d files, want 1", deletedCount)
	}

	// 古いファイルが削除されたことを確認
	if _, err := os.Stat(oldFilename); !os.IsNotExist(err) {
		t.Error("CleanupOldChanges() did not delete old file")
	}

	// 新しいファイルは残っていることを確認
	newFilename := filepath.Join(cs.changesPath, changeFileName(newSessionID))
	if _, err := os.Stat(newFilename); os.IsNotExist(err) {
		t.Error("CleanupOldChanges() deleted new file")
	}
}

func TestSessionInfo_FilesChanged(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	sessionID := "test-session-files"

	// 同じファイルへの複数の変更
	for i := 0; i < 3; i++ {
		change := ChangeRecordInput{
			FilePath:    "/tmp/test.txt",
			Timestamp:   time.Now(),
			Tool:        "write_file",
			Description: "Test change",
		}
		if err := cs.AppendChange(sessionID, change); err != nil {
			t.Fatalf("AppendChange() error = %v", err)
		}
	}

	// 別のファイルへの変更
	change2 := ChangeRecordInput{
		FilePath:    "/tmp/test2.txt",
		Timestamp:   time.Now(),
		Tool:        "str_replace",
		Description: "Test change 2",
	}
	if err := cs.AppendChange(sessionID, change2); err != nil {
		t.Fatalf("AppendChange() error = %v", err)
	}

	// セッション情報取得
	sessions, err := cs.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}

	// 該当セッションを探す
	var sessionInfo *SessionInfo
	for _, s := range sessions {
		if s.SessionID == sessionID {
			sessionInfo = &s
			break
		}
	}

	if sessionInfo == nil {
		t.Fatal("Session not found in ListSessions()")
	}

	// FilesChangedマップを確認
	if len(sessionInfo.FilesChanged) != 2 {
		t.Errorf("SessionInfo.FilesChanged has %d files, want 2", len(sessionInfo.FilesChanged))
	}

	if sessionInfo.FilesChanged["/tmp/test.txt"] != 3 {
		t.Errorf("SessionInfo.FilesChanged['/tmp/test.txt'] = %d, want 3", sessionInfo.FilesChanged["/tmp/test.txt"])
	}

	if sessionInfo.FilesChanged["/tmp/test2.txt"] != 1 {
		t.Errorf("SessionInfo.FilesChanged['/tmp/test2.txt'] = %d, want 1", sessionInfo.FilesChanged["/tmp/test2.txt"])
	}
}
