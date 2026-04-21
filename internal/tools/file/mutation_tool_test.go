package file

import (
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
