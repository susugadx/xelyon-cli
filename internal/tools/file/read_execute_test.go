package file

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteReadFile_Normal(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	output := ExecuteReadFile(testFile, 0, 0)
	if !strings.Contains(output, "line1") {
		t.Errorf("Expected content to contain 'line1', got: %s", output)
	}
}

func TestExecuteReadFile_EmptyPath(t *testing.T) {
	output := ExecuteReadFile("", 0, 0)
	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("Expected 'path is empty' error, got: %s", output)
	}
}

func TestExecuteReadFile_NonExistent(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	output := ExecuteReadFile(nonExistentFile, 0, 0)
	if !strings.Contains(output, "Error reading file") {
		t.Errorf("Expected 'Error reading file', got: %s", output)
	}
}

func TestExecuteReadFile_PathTraversal(t *testing.T) {
	output := ExecuteReadFile("../../../etc/passwd", 0, 0)
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteReadFile_QuietModeSuppressesStdout(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "quiet.txt", "line1\nline2")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	restoreQuiet := common.PushQuietMode()
	defer restoreQuiet()

	result := ExecuteReadFile(filepath.Join(tmpDir, "quiet.txt"), 0, 0)
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected no stdout in quiet mode, got %q", string(data))
	}
	if !strings.Contains(result, "1: line1") {
		t.Fatalf("ExecuteReadFile() result = %q", result)
	}
}

func TestExecuteReadFiles_QuietModeSuppressesStdout(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "file1.txt", "hello from file1")
	testutil.CreateTempFile(t, tmpDir, "file2.txt", "hello from file2")

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error = %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	restoreQuiet := common.PushQuietMode()
	defer restoreQuiet()

	result := ExecuteReadFiles([]string{
		filepath.Join(tmpDir, "file1.txt"),
		filepath.Join(tmpDir, "file2.txt"),
	})
	_ = w.Close()

	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if strings.TrimSpace(string(data)) != "" {
		t.Fatalf("expected no stdout in quiet mode, got %q", string(data))
	}
	if !strings.Contains(result, "📄 File: ") {
		t.Fatalf("ExecuteReadFiles() result = %q", result)
	}
}

func TestExecuteReadFile_BinaryFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	binFile := filepath.Join(tmpDir, "sample.bin")
	if err := os.WriteFile(binFile, []byte{0x41, 0x00, 0x42, 0x43}, 0644); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}

	output := ExecuteReadFile(binFile, 0, 0)
	if !strings.Contains(output, "appears to be a binary file") {
		t.Fatalf("expected binary-file error, got: %s", output)
	}
}
