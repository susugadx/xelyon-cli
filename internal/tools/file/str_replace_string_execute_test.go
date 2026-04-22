package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteStrReplace_ExactMatch(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "line1\nline2\nline3")

	output, err := executeStrReplaceForTest(testFile, "line2", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "line1\nREPLACED\nline3")
}

func TestExecuteStrReplace_ReturnsLineNumbers(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.go")
	content := "line1\nline2\nTARGET\nline4\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result, err := executeStrReplaceForTest(file, "TARGET", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(result, "lines 3-3") {
		t.Fatalf("expected line number in result, got: %s", result)
	}
}

func TestExecuteStrReplace_MultipleMatches(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo\nbaz")

	output, err := executeStrReplaceForTest(testFile, "foo", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: old_str appears 2 times") {
		t.Errorf("Expected multiple matches error, got: %s", output)
	}
}

func TestExecuteStrReplace_NotFound(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "line1\nline2\nline3")

	output, err := executeStrReplaceForTest(testFile, "nonexistent", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: old_str not found") {
		t.Errorf("Expected 'not found' error, got: %s", output)
	}
}

func TestExecuteStrReplace_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "line1\nline2\nline3"
	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	output, err := executeStrReplaceForTest(testFile, "line2", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not error on cancel: %v", err)
	}
	if !strings.Contains(output, "[CANCELLED]") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, originalContent)
}

func TestExecuteStrReplace_EmptyPath(t *testing.T) {
	output, err := executeStrReplaceForTest("", "old", "new", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: path is required") {
		t.Errorf("Expected 'path is required' error, got: %s", output)
	}
}

func TestExecuteStrReplace_PathTraversal(t *testing.T) {
	setupTestEnvironment(t)
	setupTestConfirm(t, true)

	output, err := executeStrReplaceForTest("../../../etc/passwd", "old", "new", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteStrReplace_EmptyOldStr(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.txt", "content")

	output, err := executeStrReplaceForTest(filepath.Join(tmpDir, "test.txt"), "", "new", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: old_str is required") {
		t.Errorf("Expected 'old_str is required' error, got: %s", output)
	}
}
