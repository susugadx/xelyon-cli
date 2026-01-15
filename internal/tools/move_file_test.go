package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteMoveFile_Normal(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "destination.txt")
	testContent := "test content for move"

	testutil.CreateTempFile(t, tmpDir, "source.txt", testContent)

	// ファイル移動
	output, backupPath, err := executeMoveFile(srcFile, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile failed: %v", err)
	}

	if !strings.Contains(output, "✅ Moved:") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// 移動元が削除されていることを確認
	testutil.AssertFileNotExists(t, srcFile)

	// 移動先にファイルが存在することを確認
	testutil.AssertFileExists(t, destFile)
	testutil.AssertFileContent(t, destFile, testContent)

	// バックアップは作成されていないべき（上書きではない）
	if backupPath != "" {
		t.Errorf("Backup path should be empty for non-overwrite move, got: %s", backupPath)
	}
}

func TestExecuteMoveFile_Overwrite(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "destination.txt")
	srcContent := "new content"
	destContent := "old content"

	testutil.CreateTempFile(t, tmpDir, "source.txt", srcContent)
	testutil.CreateTempFile(t, tmpDir, "destination.txt", destContent)

	// ファイル移動（上書き）
	output, backupPath, err := executeMoveFile(srcFile, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile failed: %v", err)
	}

	if !strings.Contains(output, "✅ Moved (overwritten):") {
		t.Errorf("Expected overwrite message, got: %s", output)
	}

	// 移動元が削除されていることを確認
	testutil.AssertFileNotExists(t, srcFile)

	// 移動先が新しい内容になっていることを確認
	testutil.AssertFileExists(t, destFile)
	testutil.AssertFileContent(t, destFile, srcContent)

	// バックアップが作成されていることを確認
	if backupPath == "" {
		t.Error("Backup path should not be empty for overwrite")
	}
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, destContent)
}

func TestExecuteMoveFile_OverwriteCancelled(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "destination.txt")
	srcContent := "new content"
	destContent := "old content"

	testutil.CreateTempFile(t, tmpDir, "source.txt", srcContent)
	testutil.CreateTempFile(t, tmpDir, "destination.txt", destContent)

	setupTestConfirm(t, false)

	// ファイル移動（上書きキャンセル）
	output, backupPath, err := executeMoveFile(srcFile, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile should not error on cancel: %v", err)
	}

	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}

	// 両ファイルが変更されていないことを確認
	testutil.AssertFileExists(t, srcFile)
	testutil.AssertFileContent(t, srcFile, srcContent)
	testutil.AssertFileExists(t, destFile)
	testutil.AssertFileContent(t, destFile, destContent)

	// バックアップは作成されていないべき
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteMoveFile_SourceNotFound(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "nonexistent.txt")
	destFile := filepath.Join(tmpDir, "destination.txt")

	// 存在しないファイルを移動しようとする
	output, _, err := executeMoveFile(srcFile, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: Source file not found") {
		t.Errorf("Expected 'Source file not found' error, got: %s", output)
	}

	// 移動先は作成されていないべき
	testutil.AssertFileNotExists(t, destFile)
}

func TestExecuteMoveFile_SourceIsDirectory(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcDir := filepath.Join(tmpDir, "srcdir")
	destFile := filepath.Join(tmpDir, "destination.txt")

	// ソースとしてディレクトリを作成
	if err := os.Mkdir(srcDir, 0755); err != nil {
		t.Fatalf("Failed to create directory: %v", err)
	}

	// ディレクトリを移動しようとする
	output, _, err := executeMoveFile(srcDir, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: Source is a directory") {
		t.Errorf("Expected directory error, got: %s", output)
	}
}

func TestExecuteMoveFile_SameFile(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "same.txt")
	testContent := "same file content"

	testutil.CreateTempFile(t, tmpDir, "same.txt", testContent)

	// 同じファイルに移動しようとする
	output, backupPath, err := executeMoveFile(testFile, testFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile should not return error: %v", err)
	}

	if !strings.Contains(output, "No operation: Source and destination are identical") {
		t.Errorf("Expected no-op message, got: %s", output)
	}

	// ファイルは変更されていないべき
	testutil.AssertFileExists(t, testFile)
	testutil.AssertFileContent(t, testFile, testContent)

	// バックアップは作成されていないべき
	if backupPath != "" {
		t.Errorf("Backup path should be empty for no-op, got: %s", backupPath)
	}
}

func TestExecuteMoveFile_DestinationDirNotExist(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	destFile := filepath.Join(tmpDir, "nonexistent_dir", "destination.txt")
	testContent := "test content"

	testutil.CreateTempFile(t, tmpDir, "source.txt", testContent)

	// 存在しないディレクトリに移動しようとする
	output, _, err := executeMoveFile(srcFile, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: Destination directory does not exist") {
		t.Errorf("Expected destination directory error, got: %s", output)
	}

	// ソースファイルは変更されていないべき
	testutil.AssertFileExists(t, srcFile)
	testutil.AssertFileContent(t, srcFile, testContent)
}

func TestExecuteMoveFile_PathTraversal(t *testing.T) {
	// 対話モードのみ無効化（ValidatePathはモックしない - 実際のセキュリティを確認）
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() { os.Unsetenv("XELYON_INTERACTIVE_CONFIRM") })
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	maliciousDest := "../../../etc/passwd"

	testutil.CreateTempFile(t, tmpDir, "source.txt", "test content")

	// パストラバーサル攻撃を試みる（移動先）
	output, _, err := executeMoveFile(srcFile, maliciousDest)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile should not return error: %v", err)
	}

	// ValidatePathでエラーになるべき
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}

	// ソースファイルは変更されていないべき
	testutil.AssertFileExists(t, srcFile)
}

func TestExecuteMoveFile_Subdirectory(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "source.txt")
	subDir := filepath.Join(tmpDir, "subdir")
	destFile := filepath.Join(subDir, "destination.txt")
	testContent := "move to subdirectory"

	testutil.CreateTempFile(t, tmpDir, "source.txt", testContent)

	// サブディレクトリ作成
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// サブディレクトリに移動
	output, _, err := executeMoveFile(srcFile, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile failed: %v", err)
	}

	if !strings.Contains(output, "✅ Moved:") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// 移動元が削除されていることを確認
	testutil.AssertFileNotExists(t, srcFile)

	// 移動先にファイルが存在することを確認
	testutil.AssertFileExists(t, destFile)
	testutil.AssertFileContent(t, destFile, testContent)
}

func TestExecuteMoveFile_Rename(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	srcFile := filepath.Join(tmpDir, "old_name.txt")
	destFile := filepath.Join(tmpDir, "new_name.txt")
	testContent := "rename test"

	testutil.CreateTempFile(t, tmpDir, "old_name.txt", testContent)

	// 同じディレクトリ内でリネーム
	output, _, err := executeMoveFile(srcFile, destFile)

	// 検証
	if err != nil {
		t.Fatalf("executeMoveFile failed: %v", err)
	}

	if !strings.Contains(output, "✅ Moved:") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// 古いファイル名が削除されていることを確認
	testutil.AssertFileNotExists(t, srcFile)

	// 新しいファイル名で存在することを確認
	testutil.AssertFileExists(t, destFile)
	testutil.AssertFileContent(t, destFile, testContent)
}
