package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestExecuteStrReplace_ExactMatch(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "line1\nline2\nline3")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := ExecuteStrReplace(testFile, "line2", "REPLACED", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "line1\nREPLACED\nline3")
}

func TestExecuteStrReplace_MultipleMatches(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo\nbaz")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := ExecuteStrReplace(testFile, "foo", "REPLACED", "", "")

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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := ExecuteStrReplace(testFile, "nonexistent", "REPLACED", "", "")

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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := ExecuteStrReplace(testFile, "line2", "REPLACED", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace should not error on cancel: %v", err)
	}
	if !strings.Contains(output, "[CANCELLED]") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, originalContent)
}

func TestExecuteStrReplace_EmptyPath(t *testing.T) {
	output, err := ExecuteStrReplace("", "old", "new", "", "")
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
	absPath, _ := filepath.Abs(filepath.Join(tmpDir, "test.txt"))
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := ExecuteStrReplace(filepath.Join(tmpDir, "test.txt"), "", "new", "", "")

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
	absPath, _ := filepath.Abs(filepath.Join(tmpDir, "test.txt"))
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := ExecuteStrReplace(filepath.Join(tmpDir, "test.txt"), "", "X\nY", "2", "4")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced lines 2-4") {
		t.Errorf("Expected line-range success message, got: %s", output)
	}
	testutil.AssertFileContent(t, filepath.Join(tmpDir, "test.txt"), "a\nX\nY\ne")
}

func TestExecuteStrReplace_StringReplace_WarnsOnDuplicateNewStr(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	// fatih/color の global writer を差し替えて、警告文が出力されたかを検証する
	var buf strings.Builder
	oldOut := color.Output
	color.Output = &buf
	t.Cleanup(func() {
		color.Output = oldOut
	})

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "alpha\nEXISTING\nomega")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	replaceOutput, err := ExecuteStrReplace(testFile, "alpha", "EXISTING", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Warning: new_str already exists in file") {
		t.Errorf("Expected warning output, got: %s", buf.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "EXISTING\nEXISTING\nomega")
}

func TestExecuteStrReplace_StringReplace_NoWarningWhenUnique(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	// fatih/color はデフォルトで stderr に出すため、stderr を捕捉する
	var output strings.Builder
	originalStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = originalStderr
	})

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "alpha\nEXISTING\nomega")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	replaceOutput, err := ExecuteStrReplace(testFile, "alpha", "NEWVALUE", "", "")

	w.Close()
	_, _ = io.Copy(&output, r)

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(output.String(), "Warning: new_str already exists in file") {
		t.Errorf("Did not expect warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "NEWVALUE\nEXISTING\nomega")
}

func TestExecuteStrReplace_LineRange_WarnsOnDuplicateOutsideRange(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	// fatih/color の global writer を差し替えて、警告文が出力されたかを検証する
	var buf strings.Builder
	oldOut := color.Output
	color.Output = &buf
	t.Cleanup(func() {
		color.Output = oldOut
	})

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "keep\nold\nold\nkeep")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	replaceOutput, err := ExecuteStrReplace(testFile, "", "keep", "2", "3")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(buf.String(), "Warning: new_str already exists outside the target range") {
		t.Errorf("Expected warning output, got: %s", buf.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced lines 2-3") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "keep\nkeep\nkeep")
}

func TestExecuteStrReplace_LineRange_NoWarningWhenUniqueOutsideRange(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, true)

	// fatih/color はデフォルトで stderr に出すため、stderr を捕捉する
	var output strings.Builder
	originalStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Failed to create pipe: %v", err)
	}
	os.Stderr = w
	t.Cleanup(func() {
		os.Stderr = originalStderr
	})

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "keep\nold\nold\nkeep")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	replaceOutput, err := ExecuteStrReplace(testFile, "", "NEWLINE", "2", "3")

	w.Close()
	_, _ = io.Copy(&output, r)

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(output.String(), "Warning: new_str already exists outside the target range") {
		t.Errorf("Did not expect warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced lines 2-3") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "keep\nNEWLINE\nkeep")
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

func TestExecuteStrReplace_GuardBlocksUnreadFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "guarded.txt")
	testutil.CreateTempFile(t, tmpDir, "guarded.txt", "line1\nline2\nline3")

	// ReadTracker はリセット済み（setupTestMocks）— ファイルは未読状態

	output, err := ExecuteStrReplace(testFile, "line2", "REPLACED", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: You must read_file before str_replace") {
		t.Errorf("Expected read guard error, got: %s", output)
	}
	// ファイルは変更されていないこと
	testutil.AssertFileContent(t, testFile, "line1\nline2\nline3")
}

func TestExecuteStrReplace_GuardAllowsAfterRead(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "allowed.txt")
	testutil.CreateTempFile(t, tmpDir, "allowed.txt", "line1\nline2\nline3")

	// read_file をシミュレート
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := ExecuteStrReplace(testFile, "line2", "REPLACED", "", "")

	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "line1\nREPLACED\nline3")
}

// ===== Batch Edits Tests =====

func TestExecuteBatchEdits_Success(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb\nccc")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	editsJSON := `[{"old_str":"aaa","new_str":"AAA"},{"old_str":"ccc","new_str":"CCC"}]`
	output, err := executeBatchEdits(testFile, editsJSON)
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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	// First edit valid, second not found
	editsJSON := `[{"old_str":"aaa","new_str":"AAA"},{"old_str":"zzz","new_str":"ZZZ"}]`
	output, err := executeBatchEdits(testFile, editsJSON)
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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	editsJSON := `[{"old_str":"foo","new_str":"baz"}]`
	output, err := executeBatchEdits(testFile, editsJSON)
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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := executeBatchEdits(testFile, "[]")
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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	output, err := executeBatchEdits(testFile, "not-json")
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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	// old_str non-empty -> single mode runs via ExecuteStrReplace, edits ignored
	output, err := ExecuteStrReplace(testFile, "aaa", "AAA", "", "")
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
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	editsJSON := `[{"old_str":"aaa","new_str":"AAA"}]`
	output, err := executeBatchEdits(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "[CANCELLED]") {
		t.Errorf("expected cancellation, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "aaa\nbbb")
}

func TestExecuteBatchEdits_SequentialApplication(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "hello world")
	absPath, _ := filepath.Abs(testFile)
	tools.GlobalReadTracker.MarkRead(absPath)

	// Edit 1 changes "hello" -> "hi", Edit 2 changes "hi world" -> "hi there"
	editsJSON := `[{"old_str":"hello","new_str":"hi"},{"old_str":"hi world","new_str":"hi there"}]`
	output, err := executeBatchEdits(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 2 edits") {
		t.Errorf("expected success, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "hi there")
}
