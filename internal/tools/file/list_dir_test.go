package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteListDir_Success(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	output := ExecuteListDir(tmpDir)

	if !strings.Contains(output, tmpDir) {
		t.Errorf("ExecuteListDir() output should contain directory path, got %v", output)
	}
	if !strings.Contains(output, "file1.txt") {
		t.Error("ExecuteListDir() output should contain 'file1.txt'")
	}
	if !strings.Contains(output, "file2.txt") {
		t.Error("ExecuteListDir() output should contain 'file2.txt'")
	}
	if !strings.Contains(output, "subdir") {
		t.Error("ExecuteListDir() output should contain 'subdir'")
	}
}

func TestExecuteListDir_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	output := ExecuteListDir(tmpDir)

	if !strings.Contains(output, tmpDir) {
		t.Errorf("ExecuteListDir() output should contain directory path, got %v", output)
	}
}

func TestExecuteListDir_DirectoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "notexist")

	output := ExecuteListDir(nonExistentDir)

	if !strings.Contains(output, "Error") {
		t.Errorf("ExecuteListDir() output = %v, should contain 'Error'", output)
	}
}

func TestExecuteListDir_FileSizes(t *testing.T) {
	tmpDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(tmpDir, "small.txt"), []byte("small"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(strings.Repeat("x", 1000)), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := ExecuteListDir(tmpDir)

	if !strings.Contains(output, "bytes") {
		t.Error("ExecuteListDir() output should contain file size in bytes")
	}
}

func TestExecuteListDir_RelativePath(t *testing.T) {
	output := ExecuteListDir(".")

	absPath, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if !strings.Contains(output, absPath) {
		t.Errorf("ExecuteListDir() output should contain absolute path, got %v", output)
	}
}
