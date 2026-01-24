package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteSearchFile_Success(t *testing.T) {
	tmpDir := t.TempDir()

	testFiles := []string{"test.go", "test.txt", "test.md"}
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	pattern := "*.go"
	output := ExecuteSearchFile(pattern, tmpDir)

	if !strings.Contains(output, "test.go") {
		t.Errorf("ExecuteSearchFile() output = %v, should contain 'test.go'", output)
	}

	if strings.Contains(output, "test.txt") {
		t.Error("ExecuteSearchFile() should not match non-Go files")
	}
}

func TestExecuteSearchFile_EmptyPattern(t *testing.T) {
	tmpDir := t.TempDir()

	output := ExecuteSearchFile("", tmpDir)

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "required") {
		t.Errorf("ExecuteSearchFile() output = %v, should contain error about required pattern", output)
	}
}

func TestExecuteSearchFile_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "*.nonexistent"
	output := ExecuteSearchFile(pattern, tmpDir)

	if !strings.Contains(output, "No files found") {
		t.Errorf("ExecuteSearchFile() output = %v, should contain 'No files found'", output)
	}
}

func TestExecuteSearchFile_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	pattern := "*.txt"
	output := ExecuteSearchFile(pattern, tmpDir)

	for _, file := range files {
		if !strings.Contains(output, file) {
			t.Errorf("ExecuteSearchFile() output should contain file %s", file)
		}
	}
}

func TestExecuteSearchFile_ExcludeGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	gitFile := filepath.Join(gitDir, "config.txt")
	if err := os.WriteFile(gitFile, []byte("git config"), 0644); err != nil {
		t.Fatalf("Failed to create git file: %v", err)
	}

	normalFile := filepath.Join(tmpDir, "normal.txt")
	if err := os.WriteFile(normalFile, []byte("normal file"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	pattern := "*.txt"
	output := ExecuteSearchFile(pattern, tmpDir)

	if !strings.Contains(output, "normal.txt") {
		t.Error("ExecuteSearchFile() should find normal.txt")
	}

	if strings.Contains(output, "config.txt") {
		t.Error("ExecuteSearchFile() should exclude files in .git directory")
	}
}

func TestExecuteSearchFile_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	nestedDir := filepath.Join(tmpDir, "level1", "level2")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	nestedFile := filepath.Join(nestedDir, "nested.go")
	if err := os.WriteFile(nestedFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	rootFile := filepath.Join(tmpDir, "root.go")
	if err := os.WriteFile(rootFile, []byte("root content"), 0644); err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	pattern := "*.go"
	output := ExecuteSearchFile(pattern, tmpDir)

	if !strings.Contains(output, "nested.go") {
		t.Error("ExecuteSearchFile() should find nested.go")
	}
	if !strings.Contains(output, "root.go") {
		t.Error("ExecuteSearchFile() should find root.go")
	}
}

func TestExecuteSearchFile_LongOutput(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 0; i < 40; i++ {
		fileName := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(fileName, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	pattern := "*.txt"
	output := ExecuteSearchFile(pattern, tmpDir)

	if strings.Contains(output, "more files") {
		lines := strings.Split(output, "\n")
		if len(lines) > 35 {
			t.Errorf("ExecuteSearchFile() output should be truncated, got %d lines", len(lines))
		}
	}
}

func TestExecuteSearchFile_ExactFileName(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("readme content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	otherFile := filepath.Join(tmpDir, "OTHER.md")
	if err := os.WriteFile(otherFile, []byte("other content"), 0644); err != nil {
		t.Fatalf("Failed to create other file: %v", err)
	}

	pattern := "README.md"
	output := ExecuteSearchFile(pattern, tmpDir)

	if !strings.Contains(output, "README.md") {
		t.Error("ExecuteSearchFile() should find README.md")
	}
	if strings.Contains(output, "OTHER.md") {
		t.Error("ExecuteSearchFile() should not find OTHER.md with exact pattern")
	}
}

func TestExecuteSearchFile_NonexistentDirectory(t *testing.T) {
	pattern := "*.txt"
	output := ExecuteSearchFile(pattern, "/nonexistent/directory/12345")

	if len(output) == 0 {
		t.Error("ExecuteSearchFile() should return some output for nonexistent directory")
	}
}

func TestExecuteSearchFile_CaseSensitivity(t *testing.T) {
	tmpDir := t.TempDir()

	lowerFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(lowerFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "test.txt"
	output := ExecuteSearchFile(pattern, tmpDir)

	if !strings.Contains(output, "test.txt") {
		t.Errorf("ExecuteSearchFile() should find test.txt, got %v", output)
	}

	patternUpper := "TEST.TXT"
	outputUpper := ExecuteSearchFile(patternUpper, tmpDir)

	if strings.Contains(outputUpper, "test.txt") {
		t.Error("ExecuteSearchFile() should be case-sensitive")
	}
}
