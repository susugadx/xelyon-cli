package file

import (
	"encoding/json"
	"fmt"
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
	if len(props) != 2 {
		t.Fatalf("expected 2 read_file parameters, got %d", len(props))
	}
	if _, ok := props["paths"]; !ok {
		t.Fatal("expected paths parameter")
	}
	if _, ok := props["symbol"]; ok {
		t.Fatal("symbol parameter should be removed")
	}
	// paths と targets は排他的なため required は未設定
	if _, hasRequired := params["required"]; hasRequired {
		t.Fatal("expected no required array (paths and targets are mutually exclusive)")
	}
	pathsParam, ok := props["paths"].(map[string]interface{})
	if !ok {
		t.Fatal("expected paths parameter schema")
	}
	if maxItems, ok := pathsParam["maxItems"].(int); !ok || maxItems != MaxReadFilesPaths {
		t.Fatalf("expected paths maxItems=%d, got %#v", MaxReadFilesPaths, pathsParam["maxItems"])
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	testutil.CreateTempFile(t, tmpDir, "test.go", "package main\nfunc main() {}")

	execCtx := tools.ExecutionContext{
		Stdin:  strings.NewReader(""),
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	t.Run("single_path_via_paths", func(t *testing.T) {
		pathsJSON, err := json.Marshal([]string{testFile})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		result, _, err := tool.Run(execCtx, map[string]string{
			"paths": string(pathsJSON),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "package main") {
			t.Fatalf("expected file content, got: %s", result)
		}
	})

	t.Run("paths_batch_normal", func(t *testing.T) {
		batchDir := t.TempDir()
		fileA := filepath.Join(batchDir, "a.go")
		fileB := filepath.Join(batchDir, "b.go")
		testutil.CreateTempFile(t, batchDir, "a.go", "package main\nfunc main() {}")
		testutil.CreateTempFile(t, batchDir, "b.go", "package util\nfunc helper() {}")

		pathsJSON, err := json.Marshal([]string{fileA, fileB})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		result, _, err := tool.Run(execCtx, map[string]string{
			"paths": string(pathsJSON),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "📄 File: "+fileA) {
			t.Fatalf("expected batch header for %s, got: %s", fileA, result)
		}
		if !strings.Contains(result, "package util") {
			t.Fatalf("expected second file content, got: %s", result)
		}
	})

	t.Run("paths_batch_max_10", func(t *testing.T) {
		batchDir := t.TempDir()
		paths := make([]string, 0, MaxReadFilesPaths)
		for i := 0; i < MaxReadFilesPaths; i++ {
			filename := fmt.Sprintf("f%d.txt", i)
			testutil.CreateTempFile(t, batchDir, filename, "line1\nline2")
			paths = append(paths, filepath.Join(batchDir, filename))
		}

		pathsJSON, err := json.Marshal(paths)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		result, _, err := tool.Run(execCtx, map[string]string{
			"paths": string(pathsJSON),
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Count(result, "📄 File: ") != MaxReadFilesPaths {
			t.Fatalf("expected %d file headers, got: %s", MaxReadFilesPaths, result)
		}
	})

	t.Run("paths_batch_full_budget", func(t *testing.T) {
		batchDir := t.TempDir()
		paths := make([]string, 0, 3)
		contentLines := make([]string, 180)
		for i := range contentLines {
			contentLines[i] = fmt.Sprintf("line%d", i+1)
		}
		content := strings.Join(contentLines, "\n")
		for i := 0; i < 3; i++ {
			filename := fmt.Sprintf("budget%d.txt", i)
			testutil.CreateTempFile(t, batchDir, filename, content)
			paths = append(paths, filepath.Join(batchDir, filename))
		}

		pathsJSON, err := json.Marshal(paths)
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		result, _, err := tool.Run(execCtx, map[string]string{
			"paths":        string(pathsJSON),
			"_full_budget": "true",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Contains(result, "lines total") {
			t.Fatalf("full budget batch should return full content, got: %s", result)
		}
		if !strings.Contains(result, "180: line180") {
			t.Fatalf("expected full content through line 180, got: %s", result)
		}
	})

	t.Run("invalid_paths_errors", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{
			"paths": "not-json",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Error: invalid paths format:") {
			t.Fatalf("expected invalid paths error, got: %s", result)
		}
	})

	t.Run("empty_paths_errors", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{
			"paths": "[]",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Error: either paths or targets is required") {
			t.Fatalf("expected empty paths error, got: %s", result)
		}
	})

	t.Run("paths_required", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Error: either paths or targets is required") {
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
