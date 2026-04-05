package file

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteStrReplace_LineRangeReplacement_Success(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
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
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("Got (%d, %d), want (%d, %d)", start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
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
