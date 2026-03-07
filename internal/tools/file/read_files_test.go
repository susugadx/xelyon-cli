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

func TestPerFileBudget(t *testing.T) {
	tests := []struct {
		n    int
		want int
	}{
		{1, MaxReadLines},
		{2, MaxReadLines},
		{3, 150},
		{5, 150},
		{6, 80},
		{10, 80},
	}
	for _, tt := range tests {
		got := perFileBudget(tt.n)
		if got != tt.want {
			t.Errorf("perFileBudget(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

// generateLines は指定行数のテスト用コンテンツを生成する
func generateLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	return strings.Join(lines, "\n")
}

func TestExecuteReadFiles_BudgetManyFiles(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 200行のファイルを6個作成（budget=80 が適用される）
	paths := make([]string, 6)
	for i := range paths {
		name := fmt.Sprintf("f%d.go", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(200))
		paths[i] = filepath.Join(tmpDir, name)
	}

	output := ExecuteReadFiles(paths)

	// 各ファイルで80行目は含まれ、81行目は含まれないことを確認
	if !strings.Contains(output, "80: line80") {
		t.Error("Expected line 80 to be present")
	}
	if strings.Contains(output, "81: line81") {
		t.Error("Expected line 81 to NOT be present (budget=80)")
	}
}

func TestExecuteReadFiles_BudgetMediumFiles(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 200行のファイルを4個作成（budget=150 が適用される）
	paths := make([]string, 4)
	for i := range paths {
		name := fmt.Sprintf("f%d.go", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(200))
		paths[i] = filepath.Join(tmpDir, name)
	}

	output := ExecuteReadFiles(paths)

	if !strings.Contains(output, "150: line150") {
		t.Error("Expected line 150 to be present")
	}
	if strings.Contains(output, "151: line151") {
		t.Error("Expected line 151 to NOT be present (budget=150)")
	}
}

func TestExecuteReadFiles_BudgetExplicitRangePriority(t *testing.T) {
	setupTestMocks(t)
	tmpDir := t.TempDir()

	// 200行のファイルを6個作成（budget=80）
	paths := make([]string, 6)
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("f%d.go", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(200))
		paths[i] = filepath.Join(tmpDir, name)
	}
	// 6番目は明示的に行範囲指定（budget より広い範囲）
	testutil.CreateTempFile(t, tmpDir, "f5.go", generateLines(200))
	paths[5] = filepath.Join(tmpDir, "f5.go") + ":1-120"

	output := ExecuteReadFiles(paths)

	// 明示指定のファイルは120行目まで含まれる（budget=80 を無視）
	if !strings.Contains(output, "120: line120") {
		t.Error("Expected explicit range file to include line 120")
	}
}
