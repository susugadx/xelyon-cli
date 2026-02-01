package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// setupTestMocks はテスト用の共通モックを設定
func setupTestMocks(t *testing.T) {
	// 対話モードを無効化（シンプルなy/n確認を使う）
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	// テスト中に .gitignore 追記のプロンプトを出さない
	os.Setenv("XELYON_SKIP_GITIGNORE_PROMPT", "1")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
		os.Unsetenv("XELYON_SKIP_GITIGNORE_PROMPT")
	})

	// confirm関数をモック（デフォルトは自動承認）
	setupTestConfirm(t, true)

	// ValidatePathをモック（テスト時はパストラバーサルチェックをスキップ）
	originalValidate := common.ValidatePath
	common.ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() {
		common.ValidatePath = originalValidate
	})
}

// setupTestConfirm はconfirm関数をモック
func setupTestConfirm(t *testing.T, result bool) {
	originalConfirm := common.SimpleConfirm
	common.SimpleConfirm = func(msg string) bool {
		return result
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
	})
}

func TestFindLatestBackup(t *testing.T) {
	dir := t.TempDir()

	// テストファイル作成
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	// バックアップファイル作成（複数）
	backup1 := testFile + ".bak.20240101_120000"
	backup2 := testFile + ".bak.20240102_120000"
	backup3 := testFile + ".bak.20240103_120000"

	for _, backup := range []string{backup1, backup2, backup3} {
		if err := os.WriteFile(backup, []byte("backup"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// 最新のバックアップを検索
	latest, err := findLatestBackup(testFile)
	if err != nil {
		t.Fatalf("findLatestBackup failed: %v", err)
	}

	// 最新のバックアップが返されるべき
	if filepath.Base(latest) != "test.txt.bak.20240103_120000" {
		t.Errorf("Expected latest backup, got %s", latest)
	}
}

func TestFindLatestBackupNoBackup(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "nobackup.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// バックアップがない場合
	_, err := findLatestBackup(testFile)
	if err == nil {
		t.Error("Expected error when no backup exists")
	}
}

func TestListBackups(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "list.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	// バックアップファイル作成
	backups := []string{
		testFile + ".bak.20240101_100000",
		testFile + ".bak.20240102_100000",
		testFile + ".bak.20240103_100000",
	}

	for _, backup := range backups {
		if err := os.WriteFile(backup, []byte("backup"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// バックアップ一覧取得
	result, err := listBackups(testFile)
	if err != nil {
		t.Fatalf("listBackups failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 backups, got %d", len(result))
	}

	// 新しい順にソートされているべき
	if filepath.Base(result[0]) != "list.txt.bak.20240103_100000" {
		t.Errorf("Expected newest backup first, got %s", result[0])
	}
}

func TestExecuteListBackups(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "execute.txt")
	if err := os.WriteFile(testFile, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}

	backup := testFile + ".bak.20240115_143000"
	if err := os.WriteFile(backup, []byte("backup content"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteListBackups(testFile)

	if result == "" {
		t.Error("Expected non-empty result")
	}

	// バックアップが含まれているべき
	if !strings.Contains(result, "execute.txt.bak.20240115_143000") {
		t.Errorf("Result should contain backup filename, got: %s", result)
	}
}

func TestExecuteListBackupsNoBackup(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "nobackup.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteListBackups(testFile)

	if !strings.Contains(result, "No backups found") {
		t.Errorf("Expected 'No backups found' message, got: %s", result)
	}
}

func TestExecuteListBackupsEmptyPath(t *testing.T) {
	result := ExecuteListBackups("")

	if !strings.Contains(result, "Error") {
		t.Errorf("Expected error for empty path, got: %s", result)
	}
}

func TestExecuteRestoreBackupEmptyPath(t *testing.T) {
	_, err := ExecuteRestoreBackup("", "")
	if err == nil {
		t.Error("Expected error for empty path")
	}
}

func TestExecuteRestoreBackupNoBackup(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "nobackup.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// バックアップがない場合
	_, err := ExecuteRestoreBackup(testFile, "")
	if err == nil {
		t.Error("Expected error when no backup exists")
	}
}

func TestExecuteRestoreBackupSpecificPath(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "restore.txt")
	backupFile := testFile + ".bak.20240115_120000"

	// オリジナルファイル
	if err := os.WriteFile(testFile, []byte("modified content"), 0644); err != nil {
		t.Fatal(err)
	}

	// バックアップファイル
	if err := os.WriteFile(backupFile, []byte("original content"), 0644); err != nil {
		t.Fatal(err)
	}

	// テストモードを有効化（自動承認、インタラクティブモード無効化）
	setupTestMocks(t)

	// 特定のバックアップから復元
	_, err := ExecuteRestoreBackup(testFile, backupFile)
	if err != nil {
		t.Fatalf("ExecuteRestoreBackup failed: %v", err)
	}

	// ファイル内容が復元されているべき
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "original content" {
		t.Errorf("Expected 'original content', got '%s'", string(content))
	}
}

func TestExecuteRestoreBackupLatest(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "latest.txt")

	// オリジナルファイル
	if err := os.WriteFile(testFile, []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}

	// バックアップファイル（複数、異なるタイムスタンプ）
	backup1 := testFile + ".bak.20240101_100000"
	backup2 := testFile + ".bak.20240115_100000" // 最新

	if err := os.WriteFile(backup1, []byte("old backup"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backup2, []byte("latest backup"), 0644); err != nil {
		t.Fatal(err)
	}

	// テストモードを有効化（自動承認、インタラクティブモード無効化）
	setupTestMocks(t)

	// backup_path未指定で最新から復元
	_, err := ExecuteRestoreBackup(testFile, "")
	if err != nil {
		t.Fatalf("ExecuteRestoreBackup failed: %v", err)
	}

	// 最新のバックアップから復元されているべき
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "latest backup" {
		t.Errorf("Expected 'latest backup', got '%s'", string(content))
	}
}

func TestRestoreBackupTool(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "tool.txt")
	backupFile := testFile + ".bak." + time.Now().Format("20060102_150405")

	// ファイル作成
	if err := os.WriteFile(testFile, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte("backup"), 0644); err != nil {
		t.Fatal(err)
	}

	// テストモードを有効化（自動承認、インタラクティブモード無効化）
	setupTestMocks(t)

	tool := &RestoreBackupTool{}

	if tool.Name() != "restore_backup" {
		t.Errorf("Expected tool name 'restore_backup', got '%s'", tool.Name())
	}

	result, change, err := tool.Run(map[string]string{
		"path":        testFile,
		"backup_path": backupFile,
	})

	if err != nil {
		t.Fatalf("Tool.Run failed: %v", err)
	}

	// restore_backupはFileChangeを返さない（復元操作のため）
	if change != nil {
		t.Error("Expected no FileChange for restore operation")
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestListBackupsTool(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "listtool.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := tools.DefaultRegistry.GetTool("list_backups")
	if tool == nil {
		t.Fatal("list_backups tool not found in registry")
	}

	if tool.Name() != "list_backups" {
		t.Errorf("Expected tool name 'list_backups', got '%s'", tool.Name())
	}

	result, change, err := tool.Run(map[string]string{
		"path": testFile,
	})

	if err != nil {
		t.Fatalf("Tool.Run failed: %v", err)
	}

	if change != nil {
		t.Error("Expected no FileChange for list operation")
	}

	// バックアップがなくてもエラーにはならない
	if result == "" {
		t.Error("Expected non-empty result")
	}
}

func TestRestoreBackupCancelled(t *testing.T) {
	dir := t.TempDir()

	testFile := filepath.Join(dir, "cancel.txt")
	backupFile := testFile + ".bak.20240115_120000"

	if err := os.WriteFile(testFile, []byte("current"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backupFile, []byte("backup"), 0644); err != nil {
		t.Fatal(err)
	}

	// テストモードを有効化（拒否、インタラクティブモード無効化）
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})
	setupTestConfirm(t, false)

	result, err := ExecuteRestoreBackup(testFile, backupFile)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// キャンセルメッセージが返されるべき
	if !strings.Contains(result, "cancelled") {
		t.Errorf("Expected cancelled message, got: %s", result)
	}

	// ファイル内容は変更されていないべき
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != "current" {
		t.Errorf("File should not be modified when cancelled, got '%s'", string(content))
	}
}
