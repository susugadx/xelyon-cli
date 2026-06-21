package readtool

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestReadFileTool_SchemaAndRun(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tool := &ReadFileTool{}

	params := tool.Parameters()
	props, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("expected properties map")
	}
	if len(props) != 3 {
		t.Fatalf("expected 3 read_file parameters, got %d", len(props))
	}
	if _, ok := props["paths"]; !ok {
		t.Fatal("expected paths parameter")
	}
	if _, ok := props["detail"]; !ok {
		t.Fatal("expected detail parameter")
	}
	if _, ok := props["symbol"]; ok {
		t.Fatal("symbol parameter should be removed")
	}
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
	detailParam, ok := props["detail"].(map[string]interface{})
	if !ok {
		t.Fatal("expected detail parameter schema")
	}
	enumValues, ok := detailParam["enum"].([]string)
	if !ok || len(enumValues) != 4 {
		t.Fatalf("expected detail enum values, got %#v", detailParam["enum"])
	}
	description, ok := detailParam["description"].(string)
	if !ok || !strings.Contains(description, "compact for locator targets or explicit path ranges") {
		t.Fatalf("expected compact restriction in detail description, got %#v", detailParam["description"])
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	testutil.CreateTempFile(t, tmpDir, "test.go", "package main\nfunc main() {}")

	execCtx := newTestToolExecContext()

	t.Run("single_path_via_paths", func(t *testing.T) {
		pathsJSON, err := json.Marshal([]string{testFile})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		result, _, err := tool.Run(execCtx, map[string]string{"paths": string(pathsJSON)})
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

		result, _, err := tool.Run(execCtx, map[string]string{"paths": string(pathsJSON)})
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

		result, _, err := tool.Run(execCtx, map[string]string{"paths": string(pathsJSON)})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if strings.Count(result, "📄 File: ") != MaxReadFilesPaths {
			t.Fatalf("expected %d file headers, got: %s", MaxReadFilesPaths, result)
		}
	})

	t.Run("full_budget_preserves_large_file_outline", func(t *testing.T) {
		batchDir := t.TempDir()
		contentLines := make([]string, 1300)
		for i := range contentLines {
			contentLines[i] = fmt.Sprintf("line%d %s", i+1, strings.Repeat("x", 900))
		}
		testFile := filepath.Join(batchDir, "budget.txt")
		testutil.CreateTempFile(t, batchDir, "budget.txt", strings.Join(contentLines, "\n"))

		pathsJSON, err := json.Marshal([]string{testFile})
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
		if !strings.Contains(result, "lines total") {
			t.Fatalf("_full_budget should keep large-file outline behavior, got: %s", result)
		}
		if strings.Contains(result, "650: line650") {
			t.Fatalf("outline output should not include middle lines, got: %s", result)
		}
	})

	t.Run("detail_overrides_full_budget", func(t *testing.T) {
		batchDir := t.TempDir()
		contentLines := make([]string, 2200)
		for i := range contentLines {
			contentLines[i] = fmt.Sprintf("line%d", i+1)
		}
		testFile := filepath.Join(batchDir, "outline.txt")
		testutil.CreateTempFile(t, batchDir, "outline.txt", strings.Join(contentLines, "\n"))

		pathsJSON, err := json.Marshal([]string{testFile})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		result, _, err := tool.Run(execCtx, map[string]string{
			"paths":        string(pathsJSON),
			"detail":       "outline",
			"_full_budget": "true",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "lines total") {
			t.Fatalf("detail=outline should win over _full_budget, got: %s", result)
		}
		if strings.Contains(result, "1500: line1500") {
			t.Fatalf("outline output should not include middle lines, got: %s", result)
		}
	})

	t.Run("detail_compact_whole_file_errors", func(t *testing.T) {
		pathsJSON, err := json.Marshal([]string{testFile})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}

		result, _, err := tool.Run(execCtx, map[string]string{
			"paths":  string(pathsJSON),
			"detail": "compact",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, `Error: detail="compact" requires locator targets or explicit path ranges`) {
			t.Fatalf("expected explicit compact whole-file error, got: %s", result)
		}
	})

	t.Run("invalid_paths_errors", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{"paths": "not-json"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, "Error: invalid paths format:") {
			t.Fatalf("expected invalid paths error, got: %s", result)
		}
	})

	t.Run("empty_paths_errors", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{"paths": "[]"})
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

	t.Run("invalid_detail_errors", func(t *testing.T) {
		result, _, err := tool.Run(execCtx, map[string]string{
			"detail": "dense",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result, `Error: invalid detail "dense"`) {
			t.Fatalf("expected invalid detail error, got: %s", result)
		}
	})
}

func TestReadFileTool_PathRequired(t *testing.T) {
	tool := &ReadFileTool{}

	result, _, err := tool.Run(newTestToolExecContext(), map[string]string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: either paths or targets is required") {
		t.Fatalf("expected error when paths is not provided, got: %s", result)
	}
}
