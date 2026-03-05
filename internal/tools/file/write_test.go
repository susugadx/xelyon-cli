package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteWriteFile_NewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	testContent := "new file content"

	output, err := ExecuteWriteFile(testFile, testContent)

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
	testutil.AssertFileContent(t, testFile, testContent)
}

func TestExecuteWriteFile_Overwrite(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	testutil.CreateTempFile(t, tmpDir, "existing.txt", "old content")

	output, err := ExecuteWriteFile(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

func TestExecuteWriteFile_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cancelled.txt")

	output, err := ExecuteWriteFile(testFile, "content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile should not error on cancel: %v", err)
	}
	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}
	testutil.AssertFileNotExists(t, testFile)
}

func TestExecuteWriteFile_EmptyPath(t *testing.T) {
	output, err := ExecuteWriteFile("", "content")
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

	output, err := ExecuteWriteFile("../../../etc/passwd", "content")
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

	output, err := ExecuteWriteFile(testFile, "content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
}

func TestExecuteWriteFile_OverwriteExistingFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readable.txt")
	testutil.CreateTempFile(t, tmpDir, "readable.txt", "old content")

	output, err := ExecuteWriteFile(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "new content")
}

func TestExecuteWriteFile_AtomicWriteNoTempFileLeft(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic.txt")
	testutil.CreateTempFile(t, tmpDir, "atomic.txt", "before")

	output, err := ExecuteWriteFile(testFile, "after")
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "after")

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read tmp directory: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".xelyon-write-") && strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temporary file should not remain after successful write: %s", e.Name())
		}
	}
}

func TestExecuteWriteFile_SuccessMessageIncludesLineCount(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")

	result, err := ExecuteWriteFile(file, "line1\nline2\nline3")
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(result, "3 lines") {
		t.Fatalf("expected line count in result, got: %s", result)
	}
}
