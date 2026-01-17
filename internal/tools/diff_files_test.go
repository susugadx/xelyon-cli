package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffFilesIdentical(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")

	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(file1, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := executeDiffFiles(file1, file2, 3)
	if err != nil {
		t.Fatalf("executeDiffFiles failed: %v", err)
	}

	if !strings.Contains(result, "identical") {
		t.Errorf("Expected 'identical' in result, got: %s", result)
	}
}

func TestDiffFilesDifferent(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")

	if err := os.WriteFile(file1, []byte("line1\nline2\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("line1\nmodified\nline3\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := executeDiffFiles(file1, file2, 3)
	if err != nil {
		t.Fatalf("executeDiffFiles failed: %v", err)
	}

	// 差分が表示されるべき
	if !strings.Contains(result, "---") || !strings.Contains(result, "+++") {
		t.Errorf("Expected diff headers in result, got: %s", result)
	}
}

func TestDiffFilesAddedLines(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")

	if err := os.WriteFile(file1, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("line1\nline2\nline3\nline4\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := executeDiffFiles(file1, file2, 3)
	if err != nil {
		t.Fatalf("executeDiffFiles failed: %v", err)
	}

	// 追加行が表示されるべき
	if !strings.Contains(result, "+") {
		t.Errorf("Expected '+' for added lines in result, got: %s", result)
	}
}

func TestDiffFilesRemovedLines(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")

	if err := os.WriteFile(file1, []byte("line1\nline2\nline3\nline4\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("line1\nline4\n"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := executeDiffFiles(file1, file2, 3)
	if err != nil {
		t.Fatalf("executeDiffFiles failed: %v", err)
	}

	// 削除行が表示されるべき
	if !strings.Contains(result, "-") {
		t.Errorf("Expected '-' for removed lines in result, got: %s", result)
	}
}

func TestDiffFilesNotExist(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "nonexistent.txt")

	if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := executeDiffFiles(file1, file2, 3)
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestDiffFilesEmptyArgs(t *testing.T) {
	_, err := executeDiffFiles("", "", 3)
	if err == nil {
		t.Error("Expected error for empty file paths")
	}
}

func TestDiffFilesTool(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "file1.txt")
	file2 := filepath.Join(dir, "file2.txt")

	if err := os.WriteFile(file1, []byte("original"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	tool := &DiffFilesTool{}

	if tool.Name() != "diff_files" {
		t.Errorf("Expected tool name 'diff_files', got '%s'", tool.Name())
	}

	result, change, err := tool.Run(map[string]string{
		"file1": file1,
		"file2": file2,
	})

	if err != nil {
		t.Fatalf("Tool.Run failed: %v", err)
	}

	// diff_filesはファイル変更しないのでchangeはnil
	if change != nil {
		t.Error("Expected no FileChange for diff_files")
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}
