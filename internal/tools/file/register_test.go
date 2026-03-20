package file

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestReadFileTool_SchemaAndRun(t *testing.T) {
	setupTestMocks(t)
	tool := &ReadFileTool{}

	params := tool.Parameters()
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}
	if len(props) != 3 {
		t.Fatalf("expected 3 read_file parameters, got %d", len(props))
	}
	if _, ok := props["path"]; !ok {
		t.Fatal("expected path parameter")
	}
	if _, ok := props["start_line"]; !ok {
		t.Fatal("expected start_line parameter")
	}
	if _, ok := props["end_line"]; !ok {
		t.Fatal("expected end_line parameter")
	}
	if _, ok := props["symbol"]; ok {
		t.Fatal("symbol parameter should be removed")
	}
	if _, ok := props["paths"]; ok {
		t.Fatal("paths parameter should be removed")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	testutil.CreateTempFile(t, tmpDir, "test.go", "package main\nfunc main() {}")

	execCtx := tools.ExecutionContext{
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	t.Run("single_path_normal", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{
			"path": testFile,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "package main") {
			t.Fatalf("expected file content, got: %s", result)
		}
	})

	t.Run("path_required", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Error: path is required") {
			t.Fatalf("expected error, got: %s", result)
		}
	})
}

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
