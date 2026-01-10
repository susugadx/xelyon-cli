package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteDeleteLines_Normal(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3\nline4\nline5"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	// 2-4行目を削除
	output, backupPath, err := executeDeleteLines(testFile, "2", "4")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted lines 2-4") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// バックアップ確認
	testutil.AssertFileExists(t, backupPath)
	testutil.AssertFileContent(t, backupPath, testContent)

	// 変更後のファイル確認（line1とline5だけ残る）
	expectedContent := "line1\nline5"
	testutil.AssertFileContent(t, testFile, expectedContent)
}

func TestExecuteDeleteLines_SingleLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	// 2行目のみ削除
	output, backupPath, err := executeDeleteLines(testFile, "2", "2")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted lines 2-2") {
		t.Errorf("Expected success message, got: %s", output)
	}

	testutil.AssertFileExists(t, backupPath)

	// line2が削除されていることを確認
	expectedContent := "line1\nline3"
	testutil.AssertFileContent(t, testFile, expectedContent)
}

func TestExecuteDeleteLines_OutOfBoundsClamping(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	// 1-999行目を削除（範囲外は自動クランプ）
	output, backupPath, err := executeDeleteLines(testFile, "1", "999")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted lines") {
		t.Errorf("Expected success message, got: %s", output)
	}

	testutil.AssertFileExists(t, backupPath)

	// 全行削除されているはず
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// 全行削除されると空またはほぼ空になる
	if len(strings.TrimSpace(string(content))) > 0 {
		t.Errorf("Expected empty or near-empty file, got: %s", string(content))
	}
}

func TestExecuteDeleteLines_InvalidStartLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	// 不正な開始行（文字列）
	output, _, err := executeDeleteLines(testFile, "abc", "3")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: Invalid start_line") {
		t.Errorf("Expected invalid start_line error, got: %s", output)
	}

	// ファイルは変更されていないべき
	testutil.AssertFileContent(t, testFile, testContent)
}

func TestExecuteDeleteLines_InvalidEndLine(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	// 不正な終了行（文字列）
	output, _, err := executeDeleteLines(testFile, "1", "xyz")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines should not return error: %v", err)
	}

	if !strings.Contains(output, "Error: Invalid end_line") {
		t.Errorf("Expected invalid end_line error, got: %s", output)
	}

	// ファイルは変更されていないべき
	testutil.AssertFileContent(t, testFile, testContent)
}

func TestExecuteDeleteLines_InvalidRange(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3\nline4\nline5"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	tests := []struct {
		name      string
		startLine string
		endLine   string
		wantErr   string
	}{
		{
			name:      "start less than 1",
			startLine: "0",
			endLine:   "3",
			wantErr:   "Error: start_line must be >= 1",
		},
		{
			name:      "end less than start",
			startLine: "5",
			endLine:   "2",
			wantErr:   "Error: end_line must be >= start_line",
		},
		{
			name:      "negative start",
			startLine: "-1",
			endLine:   "3",
			wantErr:   "Error: start_line must be >= 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, _, err := executeDeleteLines(testFile, tt.startLine, tt.endLine)

			if err != nil {
				t.Fatalf("executeDeleteLines should not return error: %v", err)
			}

			if !strings.Contains(output, tt.wantErr) {
				t.Errorf("Expected error '%s', got: %s", tt.wantErr, output)
			}

			// ファイルは変更されていないべき
			testutil.AssertFileContent(t, testFile, testContent)
		})
	}
}

func TestExecuteDeleteLines_UserCancelled(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, false)

	// 削除実行（キャンセル）
	output, backupPath, err := executeDeleteLines(testFile, "1", "2")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines should not error on cancel: %v", err)
	}

	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}

	// ファイルは変更されていないべき
	testutil.AssertFileContent(t, testFile, testContent)

	// バックアップは作成されていないべき
	if backupPath != "" {
		t.Errorf("Backup path should be empty on cancel, got: %s", backupPath)
	}
}

func TestExecuteDeleteLines_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	testutil.CreateTempFile(t, tmpDir, "empty.txt", "")

	setupTestConfirm(t, true)

	// 空ファイルから行削除を試みる
	output, _, err := executeDeleteLines(testFile, "1", "1")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines should not return error: %v", err)
	}

	// 空ファイルは空文字列の1行として扱われるため、削除は成功する
	if !strings.Contains(output, "✅ Deleted lines") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

func TestExecuteDeleteLines_FirstLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3\nline4\nline5"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	// 最初の3行を削除
	output, backupPath, err := executeDeleteLines(testFile, "1", "3")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted lines 1-3") {
		t.Errorf("Expected success message, got: %s", output)
	}

	testutil.AssertFileExists(t, backupPath)

	// line4とline5だけ残る
	expectedContent := "line4\nline5"
	testutil.AssertFileContent(t, testFile, expectedContent)
}

func TestExecuteDeleteLines_LastLines(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "line1\nline2\nline3\nline4\nline5"

	testutil.CreateTempFile(t, tmpDir, "test.txt", testContent)

	setupTestConfirm(t, true)

	// 最後の2行を削除
	output, backupPath, err := executeDeleteLines(testFile, "4", "5")

	// 検証
	if err != nil {
		t.Fatalf("executeDeleteLines failed: %v", err)
	}

	if !strings.Contains(output, "✅ Deleted lines 4-5") {
		t.Errorf("Expected success message, got: %s", output)
	}

	testutil.AssertFileExists(t, backupPath)

	// line1-3だけ残る
	expectedContent := "line1\nline2\nline3"
	testutil.AssertFileContent(t, testFile, expectedContent)
}
