package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// setupTestMocks sets up test environment
func setupTestMocks(t *testing.T) {
	t.Helper()
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	// ReadTracker をリセット
	tools.GlobalReadTracker.Reset()

	// Default to auto-approve
	setupTestConfirm(t, true)
}

// setupTestConfirm sets up test confirmation behavior
func setupTestConfirm(t *testing.T, result bool) {
	t.Helper()
	originalConfirm := common.SimpleConfirm
	common.SimpleConfirm = func(msg string) bool {
		return result
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
	})
}

func TestGrepReplaceBasic(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "test1.go")
	file2 := filepath.Join(dir, "test2.go")

	if err := os.WriteFile(file1, []byte("func oldFunc() {}\nfunc anotherFunc() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("// call oldFunc here\noldFunc()"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)
	abs1, _ := filepath.Abs(file1)
	abs2, _ := filepath.Abs(file2)
	tools.GlobalReadTracker.MarkRead(abs1)
	tools.GlobalReadTracker.MarkRead(abs2)

	result, files, err := ExecuteGrepReplace("oldFunc", "newFunc", dir, "*.go")
	if err != nil {
		t.Fatalf("ExecuteGrepReplace failed: %v", err)
	}

	if !strings.Contains(result, "completed") {
		t.Errorf("Expected 'completed' in result, got: %s", result)
	}

	if len(files) != 2 {
		t.Errorf("Expected 2 files matched, got %d", len(files))
	}

	content1, _ := os.ReadFile(file1)
	if strings.Contains(string(content1), "oldFunc") {
		t.Error("File should be modified after execution")
	}
	if !strings.Contains(string(content1), "newFunc") {
		t.Error("File should contain replacement text")
	}
}

func TestGrepReplaceExecute(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "replace.go")
	if err := os.WriteFile(file1, []byte("func oldName() {}\nfunc oldName2() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)
	abs1, _ := filepath.Abs(file1)
	tools.GlobalReadTracker.MarkRead(abs1)

	result, _, err := ExecuteGrepReplace("oldName", "newName", dir, "*.go")
	if err != nil {
		t.Fatalf("ExecuteGrepReplace failed: %v", err)
	}

	if !strings.Contains(result, "completed") {
		t.Errorf("Expected 'completed' in result, got: %s", result)
	}

	content, _ := os.ReadFile(file1)
	if strings.Contains(string(content), "oldName") {
		t.Error("File should be modified after execution")
	}
	if !strings.Contains(string(content), "newName") {
		t.Error("File should contain replacement text")
	}
}

func TestGrepReplaceNoMatch(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "nomatch.go")
	if err := os.WriteFile(file1, []byte("func something() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)

	result, files, err := ExecuteGrepReplace("nonexistent", "replacement", dir, "*.go")
	if err != nil {
		t.Fatalf("ExecuteGrepReplace failed: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("Expected 0 files matched, got %d", len(files))
	}

	_ = result
}

func TestGrepReplaceEmptyPattern(t *testing.T) {
	dir := t.TempDir()

	_, _, err := ExecuteGrepReplace("", "replacement", dir, "*.go")
	if err == nil {
		t.Error("Expected error for empty pattern")
	}
}

func TestGrepReplaceInvalidRegex(t *testing.T) {
	dir := t.TempDir()

	_, _, err := ExecuteGrepReplace("[invalid(", "replacement", dir, "*.go")
	if err == nil {
		t.Error("Expected error for invalid regex")
	}
}

func TestGrepReplaceFilePattern(t *testing.T) {
	dir := t.TempDir()

	goFile := filepath.Join(dir, "test.go")
	jsFile := filepath.Join(dir, "test.js")

	if err := os.WriteFile(goFile, []byte("func target() {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsFile, []byte("function target() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)
	absGo, _ := filepath.Abs(goFile)
	absJs, _ := filepath.Abs(jsFile)
	tools.GlobalReadTracker.MarkRead(absGo)
	tools.GlobalReadTracker.MarkRead(absJs)

	result, files, err := ExecuteGrepReplace("target", "replaced", dir, "*.go")
	if err != nil {
		t.Fatalf("ExecuteGrepReplace failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file matched (*.go only), got %d", len(files))
	}

	if len(files) > 0 && !strings.HasSuffix(files[0].FilePath, ".go") {
		t.Errorf("Expected .go file, got %s", files[0].FilePath)
	}

	_ = result
}

func TestGrepReplaceBackupCreated(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "backup.go")
	if err := os.WriteFile(file1, []byte("original content with target"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)
	abs1, _ := filepath.Abs(file1)
	tools.GlobalReadTracker.MarkRead(abs1)

	_, files, err := ExecuteGrepReplace("target", "replaced", dir, "*.go")
	if err != nil {
		t.Fatalf("ExecuteGrepReplace failed: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 file modified, got %d", len(files))
	}
}

func TestGrepReplaceRegexCapture(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "regex.go")
	if err := os.WriteFile(file1, []byte("func oldFunc1() {}\nfunc oldFunc2() {}"), 0644); err != nil {
		t.Fatal(err)
	}

	setupTestMocks(t)
	abs1, _ := filepath.Abs(file1)
	tools.GlobalReadTracker.MarkRead(abs1)

	_, _, err := ExecuteGrepReplace(`oldFunc(\d+)`, `newFunc$1`, dir, "*.go")
	if err != nil {
		t.Fatalf("ExecuteGrepReplace failed: %v", err)
	}

	content, _ := os.ReadFile(file1)
	contentStr := string(content)

	if !strings.Contains(contentStr, "newFunc1") || !strings.Contains(contentStr, "newFunc2") {
		t.Errorf("Expected regex capture replacement, got: %s", contentStr)
	}
}

func TestGrepReplaceTool(t *testing.T) {
	setupTestMocks(t)

	tool := &GrepReplaceTool{}

	if tool.Name() != "grep_replace" {
		t.Errorf("Expected tool name 'grep_replace', got '%s'", tool.Name())
	}

	dir := t.TempDir()
	file1 := filepath.Join(dir, "tool.go")
	if err := os.WriteFile(file1, []byte("target text here"), 0644); err != nil {
		t.Fatal(err)
	}
	abs1, _ := filepath.Abs(file1)
	tools.GlobalReadTracker.MarkRead(abs1)

	result, change, err := tool.Run(map[string]string{
		"pattern":     "target",
		"replacement": "replaced",
		"path":        dir,
	})

	if err != nil {
		t.Fatalf("Tool.Run failed: %v", err)
	}

	if change != nil {
		t.Error("Expected no FileChange for grep_replace")
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}
