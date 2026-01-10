package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteWriteFile_NewFile(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	testContent := "new file content"

	// 新規ファイル作成
	output, backupPath, err := executeWriteFile(testFile, testContent)

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile failed: %v", err)
	}

	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイルが作成されていることを確認
	testutil.AssertFileExists(t, testFile)
	testutil.AssertFileContent(t, testFile, testContent)

	// 新規作成なのでバックアップはないべき
	if backupPath != "" {
		t.Errorf("Backup path should be empty for new file, got: %s", backupPath)
	}
}

func TestExecuteWriteFile_Overwrite(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	oldContent := "old content"
	newContent := "new content"

	// 既存ファイル作成
	testutil.CreateTempFile(t, tmpDir, "existing.txt", oldContent)

	// 上書き
	output, backupPath, err := executeWriteFile(testFile, newContent)

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile failed: %v", err)
	}

	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイルが新しい内容になっていることを確認
	testutil.AssertFileContent(t, testFile, newContent)

	// バックアップが作成されていることを確認
	if backupPath == "" {
		t.Error("Backup path should not be empty for overwrite")
	}
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, oldContent)
}

func TestExecuteWriteFile_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cancelled.txt")
	testContent := "content that should not be written"

	// 書き込み実行（キャンセル）
	output, backupPath, err := executeWriteFile(testFile, testContent)

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile should not error on cancel: %v", err)
	}

	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}

	// ファイルが作成されていないことを確認
	testutil.AssertFileNotExists(t, testFile)

	// バックアップは作成されていないべき
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteWriteFile_EmptyPath(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	// 空パス
	output, _, err := executeWriteFile("", "content")

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("Expected 'path is empty' error, got: %s", output)
	}
}

func TestExecuteWriteFile_PathTraversal(t *testing.T) {
	setupTestConfirm(t, true)

	// パストラバーサル攻撃を試みる
	maliciousPath := "../../../etc/passwd"

	// 書き込み実行
	output, _, err := executeWriteFile(maliciousPath, "malicious content")

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile should not return error: %v", err)
	}

	// ValidatePathでエラーになるべき
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteWriteFile_CreateDirectory(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "nested", "file.txt")
	testContent := "content in nested directory"

	// 親ディレクトリが存在しない状態でファイル作成
	output, _, err := executeWriteFile(testFile, testContent)

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile failed: %v", err)
	}

	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイルが作成されていることを確認
	testutil.AssertFileExists(t, testFile)
	testutil.AssertFileContent(t, testFile, testContent)

	// 親ディレクトリも自動作成されていることを確認
	parentDir := filepath.Join(tmpDir, "subdir", "nested")
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Error("Parent directory should be created automatically")
	}
}

func TestExecuteWriteFile_MultilineContent(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiline.txt")
	testContent := "line1\nline2\nline3\nline4\nline5"

	// マルチライン書き込み
	output, _, err := executeWriteFile(testFile, testContent)

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile failed: %v", err)
	}

	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイル内容確認
	testutil.AssertFileContent(t, testFile, testContent)
}

func TestExecuteWriteFile_FilePermissions(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "perms.txt")
	testContent := "test permissions"

	// ファイル作成
	output, _, err := executeWriteFile(testFile, testContent)

	// 検証
	if err != nil {
		t.Fatalf("executeWriteFile failed: %v", err)
	}

	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイルパーミッションが0644であることを確認
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Mode().Perm() != 0644 {
		t.Errorf("File permission should be 0644, got: %o", info.Mode().Perm())
	}
}
