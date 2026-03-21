package file

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestStrReplace_LineRangeBasic(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "line1\nline2\nline3\nline4\nline5")

	// Replace lines 2-3 with new content
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

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb\nccc")

	// Replace a single line (line 2)
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

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "line1\nline2\nline3")

	tests := []struct {
		name      string
		startLine string
		endLine   string
		errSubstr string
	}{
		{
			name:      "start_line_beyond_file",
			startLine: "10",
			endLine:   "12",
			errSubstr: "start_line is out of range",
		},
		{
			name:      "end_line_beyond_file",
			startLine: "1",
			endLine:   "100",
			errSubstr: "end_line is out of range",
		},
		{
			name:      "end_less_than_start",
			startLine: "3",
			endLine:   "1",
			errSubstr: "end_line must be >= start_line",
		},
		{
			name:      "zero_start_line",
			startLine: "0",
			endLine:   "2",
			errSubstr: "start_line must be >= 1",
		},
		{
			name:      "negative_start_line",
			startLine: "-1",
			endLine:   "2",
			errSubstr: "start_line must be >= 1",
		},
		{
			name:      "only_start_no_end",
			startLine: "2",
			endLine:   "",
			errSubstr: "both start_line and end_line are required",
		},
		{
			name:      "only_end_no_start",
			startLine: "",
			endLine:   "3",
			errSubstr: "both start_line and end_line are required",
		},
		{
			name:      "non_numeric_start",
			startLine: "abc",
			endLine:   "3",
			errSubstr: "invalid line range",
		},
		{
			name:      "non_numeric_end",
			startLine: "1",
			endLine:   "xyz",
			errSubstr: "invalid line range",
		},
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

	// Verify file was not modified by any of the error cases
	testutil.AssertFileContent(t, testFile, "line1\nline2\nline3")
}

func TestStrReplace_NormalizedWhitespaceMatch(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	// File content uses tabs for indentation
	originalContent := "package main\n\nfunc hello() {\n\tfmt.Println(\"hello\")\n}\n"
	testutil.CreateTempFile(t, tmpDir, "test.go", originalContent)

	// old_str uses spaces instead of tabs (whitespace mismatch)
	result, err := executeStrReplaceForTest(testFile, "func hello() {\n    fmt.Println(\"hello\")\n}", "func hello() {\n\tfmt.Println(\"world\")\n}", "", "")
	if err != nil {
		t.Fatalf("executeStrReplaceForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully replaced") {
		t.Errorf("expected success via normalized whitespace matching, got: %s", result)
	}
	if !strings.Contains(result, "normalized whitespace") {
		t.Errorf("expected 'normalized whitespace' in result, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc hello() {\n\tfmt.Println(\"world\")\n}\n")
}

func TestStrReplace_NormalizedWhitespaceMatch_ExtraSpaces(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	// File has 4-space indentation
	testutil.CreateTempFile(t, tmpDir, "test.txt", "start\n    indented line\nend")

	// old_str uses 2-space indentation (different whitespace)
	result, err := executeStrReplaceForTest(testFile, "  indented line", "  replaced line", "", "")
	if err != nil {
		t.Fatalf("executeStrReplaceForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully replaced") {
		t.Errorf("expected success via normalized match, got: %s", result)
	}
}
