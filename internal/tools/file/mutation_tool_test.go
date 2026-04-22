package file

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestStrReplaceToolRun_FileChangeOnlyOnAppliedEdit(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tool := &StrReplaceTool{}
	execCtx := newTestToolExecContext()

	t.Run("error", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":    testFile,
			"old_str": "foo",
			"new_str": "baz",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(result, "Error:") {
			t.Fatalf("expected error result, got: %s", result)
		}
		assertNoFileChange(t, change)
	})

	t.Run("cancelled", func(t *testing.T) {
		setupTestConfirm(t, false)
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		testutil.CreateTempFile(t, tmpDir, "test.txt", "hello")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":    testFile,
			"old_str": "hello",
			"new_str": "hi",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.HasPrefix(result, "[CANCELLED]") {
			t.Fatalf("expected cancelled result, got: %s", result)
		}
		assertNoFileChange(t, change)
	})

	t.Run("success", func(t *testing.T) {
		setupTestConfirm(t, true)
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "test.txt")
		testutil.CreateTempFile(t, tmpDir, "test.txt", "hello")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":    testFile,
			"old_str": "hello",
			"new_str": "hi",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully replaced") {
			t.Fatalf("expected success result, got: %s", result)
		}
		assertHasFileChange(t, change)
	})

	t.Run("batch success", func(t *testing.T) {
		setupTestConfirm(t, true)
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "batch.txt")
		testutil.CreateTempFile(t, tmpDir, "batch.txt", "hello")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":  testFile,
			"edits": `[{"old_str":"hello","new_str":"hi"}]`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully applied 1 edits") {
			t.Fatalf("expected batch success result, got: %s", result)
		}
		assertHasFileChange(t, change)
		if change.LinesAdded != 1 || change.LinesRemoved != 1 {
			t.Fatalf("expected line stats +1/-1, got +%d/-%d", change.LinesAdded, change.LinesRemoved)
		}
	})

	t.Run("batch success uses final diff line stats when preview is enabled", func(t *testing.T) {
		setupTestConfirm(t, true)
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "batch-diff.txt")
		testutil.CreateTempFile(t, tmpDir, "batch-diff.txt", "x")
		var stdout bytes.Buffer
		execCtxWithPreview := newTestToolExecContext()
		execCtxWithPreview.Stdout = &stdout

		result, change, err := tool.Run(execCtxWithPreview, map[string]string{
			"path":  testFile,
			"edits": `[{"old_str":"x","new_str":"x\ny"},{"old_str":"x\ny","new_str":"z"}]`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully applied 2 edits") {
			t.Fatalf("expected batch success result, got: %s", result)
		}
		assertHasFileChange(t, change)
		if change.LinesAdded != 1 || change.LinesRemoved != 1 {
			t.Fatalf("expected final diff line stats +1/-1, got +%d/-%d", change.LinesAdded, change.LinesRemoved)
		}
	})

	t.Run("batch success uses lightweight line stats when stdout is suppressed", func(t *testing.T) {
		setupTestConfirm(t, true)
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "batch-diff-suppressed.txt")
		testutil.CreateTempFile(t, tmpDir, "batch-diff-suppressed.txt", "x")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":  testFile,
			"edits": `[{"old_str":"x","new_str":"x\ny"},{"old_str":"x\ny","new_str":"z"}]`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully applied 2 edits") {
			t.Fatalf("expected batch success result, got: %s", result)
		}
		assertHasFileChange(t, change)
		if change.LinesAdded != 3 || change.LinesRemoved != 3 {
			t.Fatalf("expected lightweight line stats +3/-3, got +%d/-%d", change.LinesAdded, change.LinesRemoved)
		}
	})

	t.Run("batch success can force exact diff line stats when stdout is suppressed", func(t *testing.T) {
		setupTestConfirm(t, true)
		t.Setenv(envBatchExactLineStats, "1")
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "batch-diff-suppressed-force-exact.txt")
		testutil.CreateTempFile(t, tmpDir, "batch-diff-suppressed-force-exact.txt", "x")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":  testFile,
			"edits": `[{"old_str":"x","new_str":"x\ny"},{"old_str":"x\ny","new_str":"z"}]`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully applied 2 edits") {
			t.Fatalf("expected batch success result, got: %s", result)
		}
		assertHasFileChange(t, change)
		if change.LinesAdded != 1 || change.LinesRemoved != 1 {
			t.Fatalf("expected forced exact line stats +1/-1, got +%d/-%d", change.LinesAdded, change.LinesRemoved)
		}
	})

	t.Run("batch success falls back to lightweight line stats when exact diff is capped", func(t *testing.T) {
		setupTestConfirm(t, true)
		originalLimit := myersDiagonalStepLimit
		originalMin := myersMinDiagonalStepLimit
		myersDiagonalStepLimit = 1
		myersMinDiagonalStepLimit = 1
		t.Cleanup(func() {
			myersDiagonalStepLimit = originalLimit
			myersMinDiagonalStepLimit = originalMin
		})

		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "batch-diff-fallback.txt")
		testutil.CreateTempFile(t, tmpDir, "batch-diff-fallback.txt", "x")
		var stdout bytes.Buffer
		execCtxWithPreview := newTestToolExecContext()
		execCtxWithPreview.Stdout = &stdout

		result, change, err := tool.Run(execCtxWithPreview, map[string]string{
			"path":  testFile,
			"edits": `[{"old_str":"x","new_str":"x\ny"},{"old_str":"x\ny","new_str":"z"}]`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully applied 2 edits") {
			t.Fatalf("expected batch success result, got: %s", result)
		}
		assertHasFileChange(t, change)
		if change.LinesAdded != 3 || change.LinesRemoved != 3 {
			t.Fatalf("expected fallback line stats +3/-3, got +%d/-%d", change.LinesAdded, change.LinesRemoved)
		}
	})

	t.Run("batch invalid json", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "batch-invalid.txt")
		testutil.CreateTempFile(t, tmpDir, "batch-invalid.txt", "hello")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":  testFile,
			"edits": `not-json`,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Error: invalid edits JSON:") {
			t.Fatalf("expected invalid json error result, got: %s", result)
		}
		assertNoFileChange(t, change)
	})

	t.Run("line range success tracks removed lines", func(t *testing.T) {
		setupTestConfirm(t, true)
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "line-range.txt")
		testutil.CreateTempFile(t, tmpDir, "line-range.txt", "a\nb\nc")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":       testFile,
			"new_str":    "x\ny",
			"start_line": "2",
			"end_line":   "3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully replaced lines 2-3") {
			t.Fatalf("expected line-range success result, got: %s", result)
		}
		assertHasFileChange(t, change)
		if change.LinesAdded != 2 || change.LinesRemoved != 2 {
			t.Fatalf("expected line stats +2/-2, got +%d/-%d", change.LinesAdded, change.LinesRemoved)
		}
	})
}

func TestWriteDeleteToolRun_FileChangeOnlyOnAppliedEdit(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	execCtx := newTestToolExecContext()

	t.Run("write_file cancelled", func(t *testing.T) {
		setupTestConfirm(t, false)
		tool := &WriteFileTool{}
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "write.txt")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":    testFile,
			"content": "hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "Cancelled by user" {
			t.Fatalf("expected cancel result, got: %s", result)
		}
		assertNoFileChange(t, change)
	})

	t.Run("write_file success", func(t *testing.T) {
		setupTestConfirm(t, true)
		tool := &WriteFileTool{}
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "write-success.txt")

		result, change, err := tool.Run(execCtx, map[string]string{
			"path":    testFile,
			"content": "hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Successfully wrote") {
			t.Fatalf("expected success result, got: %s", result)
		}
		assertHasFileChange(t, change)
	})

	t.Run("delete_file cancelled", func(t *testing.T) {
		setupTestConfirm(t, false)
		tool := &DeleteFileTool{}
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "delete.txt")
		testutil.CreateTempFile(t, tmpDir, "delete.txt", "hello")

		result, change, err := tool.Run(execCtx, map[string]string{"path": testFile})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != "Cancelled by user" {
			t.Fatalf("expected cancel result, got: %s", result)
		}
		assertNoFileChange(t, change)
	})

	t.Run("delete_file success", func(t *testing.T) {
		setupTestConfirm(t, true)
		tool := &DeleteFileTool{}
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "delete-success.txt")
		testutil.CreateTempFile(t, tmpDir, "delete-success.txt", "hello")

		result, change, err := tool.Run(execCtx, map[string]string{"path": testFile})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "✅ Deleted:") {
			t.Fatalf("expected success result, got: %s", result)
		}
		assertHasFileChange(t, change)
	})
}
