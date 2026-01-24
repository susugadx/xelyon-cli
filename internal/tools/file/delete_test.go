package file

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

	output, backupPath, err := ExecuteDeleteFile(testFile)

	if err != nil {
		t.Fatalf("ExecuteDeleteFile failed: %v", err)
	}
	if !strings.Contains(output, "✅ Deleted:") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileNotExists(t, testFile)
	if backupPath == "" {
		t.Error("Backup path should not be empty")
	}
	testutil.AssertFileExists(t, backupPath)
}

func TestExecuteDeleteFile_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "content")

	output, backupPath, err := ExecuteDeleteFile(testFile)

	if err != nil {
		t.Fatalf("ExecuteDeleteFile should not error on cancel: %v", err)
	}
	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteDeleteFile_NonExistentFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	output, _, err := ExecuteDeleteFile(nonExistentFile)

	if err != nil {
		t.Fatalf("ExecuteDeleteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: File not found") {
		t.Errorf("Expected 'File not found' error, got: %s", output)
	}
}

func TestExecuteDeleteFile_Directory(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()

	output, _, err := ExecuteDeleteFile(tmpDir)

	if err != nil {
		t.Fatalf("ExecuteDeleteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: Cannot delete directory") {
		t.Errorf("Expected directory error, got: %s", output)
	}
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Directory should not be deleted")
	}
}

func TestExecuteDeleteFile_PathTraversal(t *testing.T) {
	setupTestConfirm(t, true)

	output, _, err := ExecuteDeleteFile("../../../etc/passwd")

	if err != nil {
		t.Fatalf("ExecuteDeleteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteDeleteFile_BackupCreated(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "important.txt")
	testContent := "critical data"
	testutil.CreateTempFile(t, tmpDir, "important.txt", testContent)

	_, backupPath, err := ExecuteDeleteFile(testFile)

	if err != nil {
		t.Fatalf("ExecuteDeleteFile failed: %v", err)
	}
	if !strings.HasPrefix(backupPath, testFile+".bak.") {
		t.Errorf("Backup path should start with %s.bak., got: %s", testFile, backupPath)
	}
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, testContent)
}
