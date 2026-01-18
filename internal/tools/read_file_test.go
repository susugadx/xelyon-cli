package tools

import (
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

	// ファイル読み込み
	output := executeReadFile(testFile, 0, 0)

	// 検証
	if !strings.Contains(output, "line1") {
		t.Errorf("Expected content to contain 'line1', got: %s", output)
	}
	if !strings.Contains(output, "line2") {
		t.Errorf("Expected content to contain 'line2', got: %s", output)
	}
	if !strings.Contains(output, "line3") {
		t.Errorf("Expected content to contain 'line3', got: %s", output)
	}
}

func TestExecuteReadFile_EmptyPath(t *testing.T) {
	// 空パス
	output := executeReadFile("", 0, 0)

	// 検証
	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("Expected 'path is empty' error, got: %s", output)
	}
}

func TestExecuteReadFile_NonExistent(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "nonexistent.txt")

	// 存在しないファイルを読み込もうとする
	output := executeReadFile(nonExistentFile, 0, 0)

	// 検証
	if !strings.Contains(output, "Error reading file") {
		t.Errorf("Expected 'Error reading file', got: %s", output)
	}
}

func TestExecuteReadFile_PathTraversal(t *testing.T) {
	// パストラバーサル攻撃を試みる
	maliciousPath := "../../../etc/passwd"

	// 読み込み実行
	output := executeReadFile(maliciousPath, 0, 0)

	// 検証（ValidatePathでエラーになるべき）
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteReadFile_LargeFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 大きなファイル（300行）を作成
	var lines []string
	for i := 0; i < 300; i++ {
		lines = append(lines, "line content")
	}
	largeContent := strings.Join(lines, "\n")
	testutil.CreateTempFile(t, tmpDir, "large.txt", largeContent)

	// 読み込み実行
	output := executeReadFile(testFile, 0, 0)

	// 検証（MaxReadLines=200行で切り詰められるべき）
	if !strings.Contains(output, "truncated") {
		t.Errorf("Expected output to be truncated, got: %s", output)
	}
	if !strings.Contains(output, "100 lines remaining") {
		t.Errorf("Expected '100 lines remaining', got: %s", output)
	}
}

func TestExecuteReadFile_EmptyFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	testutil.CreateTempFile(t, tmpDir, "empty.txt", "")

	// 空ファイル読み込み
	output := executeReadFile(testFile, 0, 0)

	// 検証（行番号付きで空行が返る: "1: \n"）
	if !strings.Contains(output, "1:") {
		t.Errorf("Expected line number output, got: %s", output)
	}
}

func TestExecuteReadFile_MultilineContent(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiline.txt")
	testContent := "First line\nSecond line\nThird line\nFourth line\nFifth line"

	testutil.CreateTempFile(t, tmpDir, "multiline.txt", testContent)

	// 読み込み実行
	output := executeReadFile(testFile, 0, 0)

	// 検証（全ての行が含まれているべき）
	expectedLines := []string{"First line", "Second line", "Third line", "Fourth line", "Fifth line"}
	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected output to contain '%s', got: %s", line, output)
		}
	}
}

func TestExecuteReadFile_LineRange(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "range.txt")
	testContent := "line1\nline2\nline3\nline4\nline5"
	testutil.CreateTempFile(t, tmpDir, "range.txt", testContent)

	// 行範囲指定で読み込み (2-4行目)
	output := executeReadFile(testFile, 2, 4)

	// 検証
	if !strings.Contains(output, "line2") {
		t.Errorf("Expected output to contain 'line2', got: %s", output)
	}
	if !strings.Contains(output, "line3") {
		t.Errorf("Expected output to contain 'line3', got: %s", output)
	}
	if !strings.Contains(output, "line4") {
		t.Errorf("Expected output to contain 'line4', got: %s", output)
	}
	// line1とline5は含まれないべき
	if strings.Contains(output, "1: line1") {
		t.Errorf("Expected output NOT to contain 'line1' at line 1, got: %s", output)
	}
	if strings.Contains(output, "5: line5") {
		t.Errorf("Expected output NOT to contain 'line5' at line 5, got: %s", output)
	}
}

func TestExecuteReadFile_LineRangeOutOfBounds(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "bounds.txt")
	testContent := "line1\nline2\nline3"
	testutil.CreateTempFile(t, tmpDir, "bounds.txt", testContent)

	// 範囲外のstart_line
	output := executeReadFile(testFile, 10, 20)
	if !strings.Contains(output, "Error: start_line 10 exceeds total lines") {
		t.Errorf("Expected error for out of bounds start_line, got: %s", output)
	}

	// end_lineが範囲外の場合は自動調整
	output2 := executeReadFile(testFile, 2, 100)
	if !strings.Contains(output2, "line2") {
		t.Errorf("Expected output to contain 'line2', got: %s", output2)
	}
	if !strings.Contains(output2, "line3") {
		t.Errorf("Expected output to contain 'line3', got: %s", output2)
	}
}
