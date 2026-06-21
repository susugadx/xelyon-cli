package mutation

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestStrReplace_LineRangeBasic(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "line1\nline2\nline3\nline4\nline5")

	result, err := executeStrReplaceForTest(testFile, "", "REPLACED_A\nREPLACED_B", "2", "3")
	if err != nil {
		t.Fatalf("executeStrReplaceForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully replaced lines 2-3") {
		t.Errorf("expected line range success message, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "line1\nREPLACED_A\nREPLACED_B\nline4\nline5")
}

func TestStrReplace_LineRangeBasic_SingleLine(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb\nccc")

	result, err := executeStrReplaceForTest(testFile, "", "BBB", "2", "2")
	if err != nil {
		t.Fatalf("executeStrReplaceForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully replaced lines 2-2") {
		t.Errorf("expected line range success message, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "aaa\nBBB\nccc")
}

func TestStrReplace_LineRangeOutOfBounds(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "line1\nline2\nline3")

	tests := []struct {
		name      string
		startLine string
		endLine   string
		errSubstr string
	}{
		{name: "start_line_beyond_file", startLine: "10", endLine: "12", errSubstr: "start_line is out of range"},
		{name: "end_line_beyond_file", startLine: "1", endLine: "100", errSubstr: "end_line is out of range"},
		{name: "end_less_than_start", startLine: "3", endLine: "1", errSubstr: "end_line must be >= start_line"},
		{name: "zero_start_line", startLine: "0", endLine: "2", errSubstr: "start_line must be >= 1"},
		{name: "negative_start_line", startLine: "-1", endLine: "2", errSubstr: "start_line must be >= 1"},
		{name: "only_start_no_end", startLine: "2", endLine: "", errSubstr: "both start_line and end_line are required"},
		{name: "only_end_no_start", startLine: "", endLine: "3", errSubstr: "both start_line and end_line are required"},
		{name: "non_numeric_start", startLine: "abc", endLine: "3", errSubstr: "invalid line range"},
		{name: "non_numeric_end", startLine: "1", endLine: "xyz", errSubstr: "invalid line range"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeStrReplaceForTest(testFile, "", "new content", tt.startLine, tt.endLine)
			if err != nil {
				t.Fatalf("executeStrReplaceForTest should not return error: %v", err)
			}
			if !strings.Contains(result, tt.errSubstr) {
				t.Errorf("expected error containing %q, got: %s", tt.errSubstr, result)
			}
		})
	}

	testutil.AssertFileContent(t, testFile, "line1\nline2\nline3")
}

func TestExecuteStrReplace_LineRange_WarnsOnDuplicateNearby(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "keep\nold\nold\nkeep")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, nil, testFile, "", "keep", "2", "3")
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
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "keep\nold\nold\nkeep")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, nil, testFile, "", "NEWLINE", "2", "3")
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

func TestExecuteStrReplace_LineRangeGoSyntaxWarning(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
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

func TestExecuteStrReplace_LineRangeOutOfRange_SummaryFirst(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

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
