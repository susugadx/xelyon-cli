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
	output := executeReadFile(testFile)

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
	output := executeReadFile("")

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
	output := executeReadFile(nonExistentFile)

	// 検証
	if !strings.Contains(output, "Error reading file") {
		t.Errorf("Expected 'Error reading file', got: %s", output)
	}
}

func TestExecuteReadFile_PathTraversal(t *testing.T) {
	// パストラバーサル攻撃を試みる
	maliciousPath := "../../../etc/passwd"

	// 読み込み実行
	output := executeReadFile(maliciousPath)

	// 検証（ValidatePathでエラーになるべき）
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteReadFile_LargeFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 大きなファイル（15000文字）を作成
	largeContent := strings.Repeat("x", 15000)
	testutil.CreateTempFile(t, tmpDir, "large.txt", largeContent)

	// 読み込み実行
	output := executeReadFile(testFile)

	// 検証（10000文字で切り詰められるべき）
	if !strings.Contains(output, "truncated") {
		t.Errorf("Expected output to be truncated, got length: %d", len(output))
	}

	// 切り詰められた内容が10000文字程度であることを確認
	if len(output) > 11000 {
		t.Errorf("Expected output to be around 10000 chars, got: %d", len(output))
	}
}

func TestExecuteReadFile_EmptyFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	testutil.CreateTempFile(t, tmpDir, "empty.txt", "")

	// 空ファイル読み込み
	output := executeReadFile(testFile)

	// 検証（空文字列が返るべき）
	if output != "" {
		t.Errorf("Expected empty output, got: %s", output)
	}
}

func TestExecuteReadFile_MultilineContent(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "multiline.txt")
	testContent := "First line\nSecond line\nThird line\nFourth line\nFifth line"

	testutil.CreateTempFile(t, tmpDir, "multiline.txt", testContent)

	// 読み込み実行
	output := executeReadFile(testFile)

	// 検証（全ての行が含まれているべき）
	expectedLines := []string{"First line", "Second line", "Third line", "Fourth line", "Fifth line"}
	for _, line := range expectedLines {
		if !strings.Contains(output, line) {
			t.Errorf("Expected output to contain '%s', got: %s", line, output)
		}
	}
}
