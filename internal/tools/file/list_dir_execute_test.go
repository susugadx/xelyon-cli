package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteListDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	output := ExecuteListDir(tmpDir, 1)

	if !strings.Contains(output, tmpDir) {
		t.Errorf("ExecuteListDir() output should contain directory path, got %v", output)
	}
	if !strings.Contains(output, "summary: depth=1, dirs=1, files=2") {
		t.Errorf("expected compact summary header, got %s", output)
	}
	if !strings.Contains(output, "dirs: subdir/") {
		t.Errorf("expected directory summary, got %s", output)
	}
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "file2.txt") {
		t.Errorf("expected representative file names, got %s", output)
	}
}

func TestExecuteListDir_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	output := ExecuteListDir(tmpDir, 1)

	if !strings.Contains(output, tmpDir) {
		t.Errorf("ExecuteListDir() output should contain directory path, got %v", output)
	}
	if !strings.Contains(output, "summary: depth=1, dirs=0, files=0") {
		t.Errorf("expected empty summary, got %s", output)
	}
}

func TestExecuteListDir_DirectoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	nonExistentDir := filepath.Join(tmpDir, "notexist")

	output := ExecuteListDir(nonExistentDir, 1)

	if !strings.Contains(output, "Error") {
		t.Errorf("ExecuteListDir() output = %v, should contain 'Error'", output)
	}
}

func TestExecuteListDir_FileSizes(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "small.txt"), []byte("small"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(strings.Repeat("x", 1000)), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := ExecuteListDir(tmpDir, 1)

	if !strings.Contains(output, "bytes") {
		t.Error("ExecuteListDir() output should contain file size in bytes")
	}
}

func TestExecuteListDir_RelativePath(t *testing.T) {
	output := ExecuteListDir(".", 1)

	absPath, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if !strings.Contains(output, absPath) {
		t.Errorf("ExecuteListDir() output should contain absolute path, got %v", output)
	}
}

func TestExecuteListDir_RejectsOutsideWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	output := ExecuteListDir("/etc", 1)
	if !strings.Contains(output, "Error") {
		t.Errorf("expected error for path outside workspace, got: %s", output)
	}
}
