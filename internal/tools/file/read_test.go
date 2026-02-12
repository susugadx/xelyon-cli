package file

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteReadFile_Normal(t *testing.T) {
	setupTestMocks(t)

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

func TestExecuteReadFile_LargeFile(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	var lines []string
	for i := 0; i < 400; i++ {
		lines = append(lines, "line content")
	}
	testutil.CreateTempFile(t, tmpDir, "large.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "large.txt"), 0, 0)
	if !strings.Contains(output, "truncated") {
		t.Errorf("Expected output to be truncated, got: %s", output)
	}
}

func TestExecuteReadFile_LineRange(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "range.txt", "line1\nline2\nline3\nline4\nline5")

	output := ExecuteReadFile(filepath.Join(tmpDir, "range.txt"), 2, 4)

	if !strings.Contains(output, "line2") || !strings.Contains(output, "line4") {
		t.Errorf("Expected output to contain lines 2-4, got: %s", output)
	}
}

func TestExecuteReadFile_StartLineOnly(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 10行のファイルを作成
	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "startonly.txt", strings.Join(lines, "\n"))

	// start_line=5, end_line=0 → 5行目から末尾まで
	output := ExecuteReadFile(filepath.Join(tmpDir, "startonly.txt"), 5, 0)

	if !strings.Contains(output, "5: line5") {
		t.Errorf("Expected output to start at line 5, got: %s", output)
	}
	if strings.Contains(output, "4: line4") {
		t.Errorf("Output should NOT contain line 4, got: %s", output)
	}
	if !strings.Contains(output, "10: line10") {
		t.Errorf("Expected output to contain last line, got: %s", output)
	}
}

func TestExecuteReadFile_EndLineOnly(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "endonly.txt", strings.Join(lines, "\n"))

	// start_line=0, end_line=3 → 1行目から3行目まで
	output := ExecuteReadFile(filepath.Join(tmpDir, "endonly.txt"), 0, 3)

	if !strings.Contains(output, "1: line1") {
		t.Errorf("Expected output to start at line 1, got: %s", output)
	}
	if !strings.Contains(output, "3: line3") {
		t.Errorf("Expected output to contain line 3, got: %s", output)
	}
	if strings.Contains(output, "4: line4") {
		t.Errorf("Output should NOT contain line 4, got: %s", output)
	}
}

func TestExecuteReadFile_StartLineLargeFile(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 300行のファイルを作成
	var lines []string
	for i := 1; i <= 300; i++ {
		lines = append(lines, fmt.Sprintf("content_line_%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "large_start.txt", strings.Join(lines, "\n"))

	// start_line=250 → 250行目から300行（= 250-300 の51行）が返る
	output := ExecuteReadFile(filepath.Join(tmpDir, "large_start.txt"), 250, 0)

	if !strings.Contains(output, "250: content_line_250") {
		t.Errorf("Expected output to start at line 250, got: %s", output)
	}
	if strings.Contains(output, "249: content_line_249") {
		t.Errorf("Output should NOT contain line 249, got: %s", output)
	}
	if !strings.Contains(output, "300: content_line_300") {
		t.Errorf("Expected output to contain last line 300, got: %s", output)
	}
}
