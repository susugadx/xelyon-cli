package file

import (
	"fmt"
	"io"
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

func TestExecuteStrReplace_DuplicateWarningForNearbyMatch(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "alpha\nEXISTING\nomega")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "alpha", "EXISTING", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output.String(), "Warning: new_str already exists near the replacement") {
		t.Errorf("Expected warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "EXISTING\nEXISTING\nomega")
}

func TestExecuteStrReplace_StringReplace_NoWarningWhenUnique(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "alpha\nEXISTING\nomega")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "alpha", "NEWVALUE", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(output.String(), "Warning: new_str already exists near the replacement") {
		t.Errorf("Did not expect warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "NEWVALUE\nEXISTING\nomega")
}

func TestExecuteStrReplace_GoSyntaxWarning(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var stdout strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	result, err := executeStrReplaceWithWritersForTest(&stdout, io.Discard, testFile, "func Build() error {", "func Build() error ", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("expected syntax warning in result, got: %s", result)
	}
	if !strings.Contains(stdout.String(), "AST syntax check found issues after replacement") {
		t.Fatalf("expected syntax warning on stdout, got: %s", stdout.String())
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc Build() error \n\treturn nil\n}\n")
}

func TestExecuteStrReplace_GoSyntaxValid(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	result, err := executeStrReplaceForTest(testFile, "return nil", "panic(\"boom\")", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("did not expect syntax warning, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc Build() error {\n\tpanic(\"boom\")\n}\n")
}

func TestExecuteStrReplace_NonGoNoValidation(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.py")
	testutil.CreateTempFile(t, tmpDir, "main.py", "def main():\n    print('ok')\n")

	result, err := executeStrReplaceForTest(testFile, "print('ok')", "if (", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("did not expect syntax warning, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "def main():\n    if (\n")
}

func TestExecuteStrReplace_NoDuplicateWarningForDistantMatch(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	lines := []string{"TARGET"}
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("middle-%d", i))
	}
	lines = append(lines, "EXISTING")
	testutil.CreateTempFile(t, tmpDir, "test.txt", strings.Join(lines, "\n"))

	result, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "TARGET", "EXISTING", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(output.String(), "Warning: new_str already exists near the replacement") {
		t.Fatalf("did not expect nearby warning, got: %s", output.String())
	}
	if !strings.Contains(result, "Successfully replaced") {
		t.Fatalf("expected success message, got: %s", result)
	}
}

func TestExecuteStrReplace_MultipleMatches_SummaryFirst(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nalpha\nfoo\nbeta\nfoo")

	output, err := executeStrReplaceForTest(testFile, "foo", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Candidates: 3 total (showing 2)") {
		t.Fatalf("expected compact candidate summary, got: %s", output)
	}
	if !strings.Contains(output, "- ... 1 more candidates") {
		t.Fatalf("expected omitted candidate summary, got: %s", output)
	}
	if strings.Contains(output, "File preview (first") {
		t.Fatalf("did not expect verbose file preview, got: %s", output)
	}
	if strings.Contains(output, "Next actions:") {
		t.Fatalf("did not expect verbose numbered next actions, got: %s", output)
	}
}

func TestExecuteStrReplace_NotFound_SummaryFirst(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "first\nsecond\nthird\nfourth")

	output, err := executeStrReplaceForTest(testFile, "missing", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Preview: 1:first | 2:second | 3:third | ... +1 more lines") {
		t.Fatalf("expected compact preview, got: %s", output)
	}
	if !strings.Contains(output, "Next: use read_file/search_code to copy the exact text") {
		t.Fatalf("expected concise next action, got: %s", output)
	}
	if strings.Contains(output, "1)") {
		t.Fatalf("did not expect numbered next actions, got: %s", output)
	}
}
