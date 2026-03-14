package file

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestStrReplaceToolRun_FileChangeOnlyOnAppliedEdit(t *testing.T) {
	setupTestMocks(t)
	tool := &StrReplaceTool{}
	execCtx := tools.ExecutionContext{
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

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
		if change != nil {
			t.Fatalf("expected nil change on error, got: %+v", change)
		}
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
		if change != nil {
			t.Fatalf("expected nil change on cancellation, got: %+v", change)
		}
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
		if change == nil {
			t.Fatal("expected non-nil change on success")
		}
	})
}
