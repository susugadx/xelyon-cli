package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteDeleteFile_Normal(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	// 削除実行
	output, backupPath, err := executeDeleteFile(testFile)

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteFile failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted:") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイルが削除されていることを確認
	testutil.AssertFileNotExists(t, testFile)

	// バックアップが作成されていることを確認
	if backupPath == "" {
		t.Error("Backup path should not be empty")
	}
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, testContent)
}

func TestExecuteDeleteFile_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "important data"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	// 削除実行
	output, backupPath, err := executeDeleteFile(testFile)

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteFile should not error on cancel: %v", err)
	}

	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}

	// ファイルが削除されていないことを確認
	testutil.AssertFileExists(t, testFile)

	// バックアップが作成されていないことを確認
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteDeleteFile_NonExistentFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	// 削除実行
	output, _, err := executeDeleteFile(nonExistentFile)

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteFile should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: File not found") {
		t.Errorf("Expected 'File not found' error, got: %s", output)
	}
}

func TestExecuteDeleteFile_Directory(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()

	// ディレクトリを削除しようとする
	output, _, err := executeDeleteFile(tmpDir)

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteFile should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: Cannot delete directory") {
		t.Errorf("Expected directory error, got: %s", output)
	}

	// ディレクトリが削除されていないことを確認
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Directory should not be deleted")
	}
}

func TestExecuteDeleteFile_PathTraversal(t *testing.T) {
	// パストラバーサルテストではモックなし（実際の検証動作を確認）
	setupTestConfirm(t, true)

	// パストラバーサル攻撃を試みる
	maliciousPath := "../../../etc/passwd"

	// 削除実行
	output, _, err := executeDeleteFile(maliciousPath)

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteFile should not return error: %v", err)
	}

	// ValidatePathでエラーになるべき
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteDeleteFile_BackupCreated(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "important.txt")
	testContent := "critical data that should be backed up"

	testutil.CreateTempFile(t, tmpDir, "important.txt", testContent)

	// 削除実行
	output, backupPath, err := executeDeleteFile(testFile)

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteFile failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted:") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// バックアップがタイムスタンプ付き.bak形式であることを確認
	if !strings.HasPrefix(backupPath, testFile+".bak.") {
		t.Errorf("Backup path should start with %s.bak., got: %s", testFile, backupPath)
	}

	// バックアップが存在し、内容が元ファイルと同じであることを確認
	if !testutil.FileExists(t, backupPath) {
		t.Fatalf("Backup file does not exist: %s", backupPath)
	}
	testutil.AssertFileContent(t, backupPath, testContent)
}

func TestExecuteDeleteFile_LargeFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 大きなファイル（100行）を作成
	var lines []string
	for i := 1; i <= 100; i++ {
		lines = append(lines, strings.Repeat("x", 80))
	}
	largeContent := strings.Join(lines, "\n")

	testutil.CreateTempFile(t, tmpDir, "large.txt", largeContent)

	// 削除実行
	output, backupPath, err := executeDeleteFile(testFile)

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteFile failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted:") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// ファイルが削除されていることを確認
	testutil.AssertFileNotExists(t, testFile)

	// バックアップが正しく作成されていることを確認
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, largeContent)
}
