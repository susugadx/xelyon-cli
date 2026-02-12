package file

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteReadFiles_Normal(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "a.go", "package main\nfunc main() {}")
	testutil.CreateTempFile(t, tmpDir, "b.go", "package util\nfunc helper() {}")

	paths := []string{
		filepath.Join(tmpDir, "a.go"),
		filepath.Join(tmpDir, "b.go"),
	}

	output := ExecuteReadFiles(paths)

	if !strings.Contains(output, "package main") {
		t.Errorf("Expected output to contain 'package main', got: %s", output)
	}
	if !strings.Contains(output, "package util") {
		t.Errorf("Expected output to contain 'package util', got: %s", output)
	}
	// ファイルヘッダーの確認
	if !strings.Contains(output, "📄 File:") {
		t.Errorf("Expected output to contain file headers, got: %s", output)
	}
}

func TestExecuteReadFiles_EmptyPaths(t *testing.T) {
	output := ExecuteReadFiles([]string{})
	if !strings.Contains(output, "Error: paths is empty") {
		t.Errorf("Expected 'paths is empty' error, got: %s", output)
	}
}

func TestExecuteReadFiles_TooManyPaths(t *testing.T) {
	paths := make([]string, 11)
	for i := range paths {
		paths[i] = fmt.Sprintf("file%d.go", i)
	}

	output := ExecuteReadFiles(paths)
	if !strings.Contains(output, "too many paths") {
		t.Errorf("Expected 'too many paths' error, got: %s", output)
	}
}

func TestExecuteReadFiles_NonExistentMixed(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "exists.go", "package exists")

	paths := []string{
		filepath.Join(tmpDir, "exists.go"),
		filepath.Join(tmpDir, "nonexistent.go"),
	}

	output := ExecuteReadFiles(paths)

	if !strings.Contains(output, "package exists") {
		t.Errorf("Expected output to contain existing file content, got: %s", output)
	}
	if !strings.Contains(output, "Error") {
		t.Errorf("Expected output to contain error for nonexistent file, got: %s", output)
	}
}

func TestExecuteReadFiles_LineRange(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "lines.txt", "line1\nline2\nline3\nline4\nline5")

	entry := filepath.Join(tmpDir, "lines.txt") + ":2-4"
	output := ExecuteReadFiles([]string{entry})

	if !strings.Contains(output, "line2") || !strings.Contains(output, "line4") {
		t.Errorf("Expected output to contain lines 2-4, got: %s", output)
	}
	if strings.Contains(output, "1: line1") {
		t.Errorf("Output should NOT contain line 1, got: %s", output)
	}
}

func TestExecuteReadFiles_StartLineOnly(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "start.txt", strings.Join(lines, "\n"))

	entry := filepath.Join(tmpDir, "start.txt") + ":8"
	output := ExecuteReadFiles([]string{entry})

	if !strings.Contains(output, "8: line8") {
		t.Errorf("Expected output to start at line 8, got: %s", output)
	}
	if strings.Contains(output, "7: line7") {
		t.Errorf("Output should NOT contain line 7, got: %s", output)
	}
}

func TestParsePath_NoRange(t *testing.T) {
	path, start, end := parsePath("internal/tools/file/read.go")
	if path != "internal/tools/file/read.go" || start != 0 || end != 0 {
		t.Errorf("Expected (internal/tools/file/read.go, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_StartOnly(t *testing.T) {
	path, start, end := parsePath("file.go:10")
	if path != "file.go" || start != 10 || end != 0 {
		t.Errorf("Expected (file.go, 10, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_StartEnd(t *testing.T) {
	path, start, end := parsePath("file.go:10-20")
	if path != "file.go" || start != 10 || end != 20 {
		t.Errorf("Expected (file.go, 10, 20), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_InvalidSuffix(t *testing.T) {
	// コロン後が数字でない場合は全体をパスとして扱う
	path, start, end := parsePath("file.go:abc")
	if path != "file.go:abc" || start != 0 || end != 0 {
		t.Errorf("Expected (file.go:abc, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}

func TestParsePath_InvalidRange(t *testing.T) {
	path, start, end := parsePath("file.go:abc-def")
	if path != "file.go:abc-def" || start != 0 || end != 0 {
		t.Errorf("Expected (file.go:abc-def, 0, 0), got (%s, %d, %d)", path, start, end)
	}
}
