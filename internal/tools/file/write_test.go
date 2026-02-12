package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteWriteFile_NewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	testContent := "new file content"

	output, backupPath, err := ExecuteWriteFile(testFile, testContent)

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
	testutil.AssertFileContent(t, testFile, testContent)
	if backupPath != "" {
		t.Errorf("Backup path should be empty for new file, got: %s", backupPath)
	}
}

func TestExecuteWriteFile_Overwrite(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	testutil.CreateTempFile(t, tmpDir, "existing.txt", "old content")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, backupPath, err := ExecuteWriteFile(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if backupPath == "" {
		t.Error("Backup path should not be empty for overwrite")
	}
}

func TestExecuteWriteFile_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cancelled.txt")

	output, backupPath, err := ExecuteWriteFile(testFile, "content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile should not error on cancel: %v", err)
	}
	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}
	testutil.AssertFileNotExists(t, testFile)
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteWriteFile_EmptyPath(t *testing.T) {
	output, _, err := ExecuteWriteFile("", "content")
	if err != nil {
		t.Fatalf("ExecuteWriteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("Expected 'path is empty' error, got: %s", output)
	}
}

func TestExecuteWriteFile_PathTraversal(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() { os.Unsetenv("XELYON_INTERACTIVE_CONFIRM") })
	setupTestConfirm(t, true)

	output, _, err := ExecuteWriteFile("../../../etc/passwd", "content")
	if err != nil {
		t.Fatalf("ExecuteWriteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteWriteFile_CreateDirectory(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "nested", "file.txt")

	output, _, err := ExecuteWriteFile(testFile, "content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
}

func TestExecuteWriteFile_GuardBlocksExistingUnread(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "guarded.txt")
	testutil.CreateTempFile(t, tmpDir, "guarded.txt", "old content")

	// ReadTracker はリセット済み（setupTestMocks）— ファイルは未読状態

	output, backupPath, err := ExecuteWriteFile(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: You must read_file before editing") {
		t.Errorf("Expected read guard error, got: %s", output)
	}
	if backupPath != "" {
		t.Errorf("Backup path should be empty when guard blocks, got: %s", backupPath)
	}
	// ファイルは変更されていないこと
	testutil.AssertFileContent(t, testFile, "old content")
}

func TestExecuteWriteFile_GuardAllowsNewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "brand_new.txt")

	// ReadTracker はリセット済み — 新規ファイルはガード対象外

	output, _, err := ExecuteWriteFile(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
	testutil.AssertFileContent(t, testFile, "new content")
}

func TestExecuteWriteFile_GuardAllowsAfterRead(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readable.txt")
	testutil.CreateTempFile(t, tmpDir, "readable.txt", "old content")

	// read_file をシミュレート
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, _, err := ExecuteWriteFile(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "new content")
}
