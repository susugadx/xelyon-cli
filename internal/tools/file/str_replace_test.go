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

func TestExecuteStrReplace_EmptyOldStr(t *testing.T) {
	setupTestMocks(t)
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

func TestExecuteStrReplace_LineRangeReplacement_Success(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.txt", "a\nb\nc\nd\ne")

	output, err := executeStrReplaceForTest(filepath.Join(tmpDir, "test.txt"), "", "X\nY", "2", "4")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced lines 2-4") {
		t.Errorf("Expected line-range success message, got: %s", output)
	}
	testutil.AssertFileContent(t, filepath.Join(tmpDir, "test.txt"), "a\nX\nY\ne")
}

func TestExecuteStrReplace_DuplicateWarningForNearbyMatch(t *testing.T) {
	setupTestMocks(t)
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

func TestExecuteStrReplace_LineRange_WarnsOnDuplicateNearby(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "keep\nold\nold\nkeep")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "", "keep", "2", "3")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output.String(), "Warning: new_str already exists near the target range") {
		t.Errorf("Expected warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced lines 2-3") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "keep\nkeep\nkeep")
}

func TestExecuteStrReplace_LineRange_NoWarningWhenUniqueOutsideRange(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "keep\nold\nold\nkeep")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "", "NEWLINE", "2", "3")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(output.String(), "Warning: new_str already exists near the target range") {
		t.Errorf("Did not expect warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced lines 2-3") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "keep\nNEWLINE\nkeep")
}

func TestExecuteStrReplace_GoSyntaxWarning(t *testing.T) {
	setupTestMocks(t)
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

func TestExecuteStrReplace_LineRangeGoSyntaxWarning(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	result, err := executeStrReplaceForTest(testFile, "", "func Build() error \n\treturn nil\n}", "3", "5")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("expected syntax warning in result, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc Build() error \n\treturn nil\n}\n")
}

func TestExecuteStrReplace_NoDuplicateWarningForDistantMatch(t *testing.T) {
	setupTestMocks(t)
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

func TestParseLineRange(t *testing.T) {
	tests := []struct {
		name      string
		startStr  string
		endStr    string
		wantStart int
		wantEnd   int
		wantErr   bool
	}{
		{"valid range 1-5", "1", "5", 1, 5, false},
		{"valid single line", "5", "5", 5, 5, false},
		{"invalid start", "abc", "5", 0, 0, true},
		{"start line zero", "0", "5", 0, 0, true},
		{"end less than start", "10", "5", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := parseLineRange(tt.startStr, tt.endStr)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if start != tt.wantStart || end != tt.wantEnd {
					t.Errorf("Got (%d, %d), want (%d, %d)", start, end, tt.wantStart, tt.wantEnd)
				}
			}
		})
	}
}

// ===== Batch Edits Tests =====

func TestExecuteBatchEdits_NormalizedWhitespaceFallback(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "func main() {\n\tfmt.Println(\"hello\")\n}"
	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// Indentation is intentionally different to trigger fallback
	editsJSON := `[{"old_str":"func main() {\n    fmt.Println(\"hello\")\n}","new_str":"func main() {\n\tfmt.Println(\"world\")\n}"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 1 edits") && !strings.Contains(output, "Summary") {
		t.Errorf("expected batch success message, got: %s", output)
	}
	// Ensure the replacement correctly handled the bounds without leaving extra characters (off-by-one fix)
	testutil.AssertFileContent(t, testFile, "func main() {\n\tfmt.Println(\"world\")\n}")
}

func TestExecuteBatchEdits_Success(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb\nccc")

	editsJSON := `[{"old_str":"aaa","new_str":"AAA"},{"old_str":"ccc","new_str":"CCC"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 2 edits") {
		t.Errorf("expected batch success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "AAA\nbbb\nCCC")
}

func TestExecuteBatchEdits_RollbackOnFailure(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "aaa\nbbb\nccc"
	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	// First edit valid, second not found
	editsJSON := `[{"old_str":"aaa","new_str":"AAA"},{"old_str":"zzz","new_str":"ZZZ"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "edits[1].old_str not found") {
		t.Errorf("expected failure on second edit, got: %s", output)
	}
	// File must be unchanged (rollback)
	testutil.AssertFileContent(t, testFile, originalContent)
}

func TestExecuteBatchEdits_AmbiguousMatch(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo")

	editsJSON := `[{"old_str":"foo","new_str":"baz"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "appears 2 times") {
		t.Errorf("expected ambiguous match error, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "foo\nbar\nfoo")
}

func TestExecuteBatchEdits_EmptyArray(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "content")

	output, err := executeBatchEditsForTest(testFile, "[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "edits array is empty") {
		t.Errorf("expected empty array error, got: %s", output)
	}
}

func TestExecuteBatchEdits_InvalidJSON(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "content")

	output, err := executeBatchEditsForTest(testFile, "not-json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "invalid edits JSON") {
		t.Errorf("expected invalid JSON error, got: %s", output)
	}
}

func TestExecuteBatchEdits_OldStrPriority(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb")

	// old_str non-empty -> single mode runs via ExecuteStrReplace, edits ignored
	output, err := executeStrReplaceForTest(testFile, "aaa", "AAA", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("expected single-mode success, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "AAA\nbbb")
}

func TestExecuteBatchEdits_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb")

	editsJSON := `[{"old_str":"aaa","new_str":"AAA"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "[CANCELLED]") {
		t.Errorf("expected cancellation, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "aaa\nbbb")
}

func TestExecuteBatchEdits_GoSyntaxWarning(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	editsJSON := `[{"old_str":"func Build() error {","new_str":"func Build() error "}]`
	result, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("expected syntax warning in result, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc Build() error \n\treturn nil\n}\n")
}

func TestExecuteBatchEdits_SequentialApplication(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "hello world")

	// Edit 1 changes "hello" -> "hi", Edit 2 changes "hi world" -> "hi there"
	editsJSON := `[{"old_str":"hello","new_str":"hi"},{"old_str":"hi world","new_str":"hi there"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 2 edits") {
		t.Errorf("expected success, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "hi there")
}

func TestExecuteStrReplace_MultipleMatches_SummaryFirst(t *testing.T) {
	setupTestMocks(t)

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

func TestExecuteStrReplace_LineRangeOutOfRange_SummaryFirst(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "a\nb\nc")

	output, err := executeStrReplaceForTest(testFile, "", "X", "5", "5")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: start_line is out of range in ") {
		t.Fatalf("expected range error with path, got: %s", output)
	}
	if !strings.Contains(output, "Next: use read_file to confirm the target range.") {
		t.Fatalf("expected concise range hint, got: %s", output)
	}
}

func TestExecuteBatchEdits_AmbiguousMatch_SummaryFirst(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo\nbaz\nfoo")

	editsJSON := `[{"old_str":"foo","new_str":"baz"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Candidates: 3 total (showing 2)") {
		t.Fatalf("expected compact candidate summary, got: %s", output)
	}
	if !strings.Contains(output, "- ... 1 more candidates") {
		t.Fatalf("expected omitted candidate summary, got: %s", output)
	}
	if strings.Contains(output, "Next actions:") {
		t.Fatalf("did not expect verbose numbered next actions, got: %s", output)
	}
}
