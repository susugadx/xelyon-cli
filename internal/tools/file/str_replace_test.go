package file

import (
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

	output, backupPath, err := ExecuteStrReplace(testFile, "line2", "REPLACED", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "line1\nREPLACED\nline3")
	if backupPath == "" {
		t.Error("Backup path should not be empty")
	}
}

func TestExecuteStrReplace_MultipleMatches(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo\nbaz")

	output, _, err := ExecuteStrReplace(testFile, "foo", "REPLACED", "", "")

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

	output, _, err := ExecuteStrReplace(testFile, "nonexistent", "REPLACED", "", "")

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

	output, backupPath, err := ExecuteStrReplace(testFile, "line2", "REPLACED", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace should not error on cancel: %v", err)
	}
	if !strings.Contains(output, "[CANCELLED]") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, originalContent)
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteStrReplace_EmptyPath(t *testing.T) {
	output, _, err := ExecuteStrReplace("", "old", "new", "", "")
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

	output, _, err := ExecuteStrReplace(filepath.Join(tmpDir, "test.txt"), "", "new", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: old_str is required") {
		t.Errorf("Expected 'old_str is required' error, got: %s", output)
	}
}

func TestExecuteStrReplace_LineRangeReplacement_Success(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.txt", "a\nb\nc\nd\ne")

	output, backupPath, err := ExecuteStrReplace(filepath.Join(tmpDir, "test.txt"), "", "X\nY", "2", "4")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced lines 2-4") {
		t.Errorf("Expected line-range success message, got: %s", output)
	}
	testutil.AssertFileContent(t, filepath.Join(tmpDir, "test.txt"), "a\nX\nY\ne")
	if backupPath == "" {
		t.Fatal("Backup path should not be empty")
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
