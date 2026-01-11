package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteInsertAfter_ExactMatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4"

	// テストファイル作成
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "Line 2"
	insertContent := "Inserted line"

	output, backupPath, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	// バックアップが作成されたことを確認
	if backupPath == "" {
		t.Error("executeInsertAfter() should create backup")
	}

	// ファイル内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := "Line 1\nLine 2\nInserted line\nLine 3\nLine 4"
	if string(gotContent) != expectedContent {
		t.Errorf("executeInsertAfter() file content = %q, want %q", string(gotContent), expectedContent)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Inserted after line") {
		t.Errorf("executeInsertAfter() output = %v, should contain 'Inserted after line'", output)
	}
}

func TestExecuteInsertAfter_NormalizedWhitespaceMatch(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	// タブとスペースを含む内容
	content := "Line 1\n\t\tLine with tabs\nLine 3"

	// テストファイル作成
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// スペースで検索（normalizeされてマッチ）
	pattern := "    Line with tabs"
	insertContent := "Inserted"

	output, backupPath, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	if backupPath == "" {
		t.Error("executeInsertAfter() should create backup")
	}

	// ファイル内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// タブが保持され、その後に挿入されることを確認
	contentStr := string(gotContent)
	if !strings.Contains(contentStr, "Inserted") {
		t.Error("executeInsertAfter() did not insert content")
	}

	lines := strings.Split(contentStr, "\n")
	if len(lines) != 4 {
		t.Errorf("executeInsertAfter() line count = %d, want 4", len(lines))
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Inserted after line") {
		t.Errorf("executeInsertAfter() output = %v", output)
	}
}

func TestExecuteInsertAfter_PatternNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3"

	// テストファイル作成
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "Nonexistent line"
	insertContent := "Inserted"

	output, backupPath, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	// バックアップは作成されない
	if backupPath != "" {
		t.Errorf("executeInsertAfter() backupPath = %v, want empty when pattern not found", backupPath)
	}

	// エラーメッセージを確認
	if !strings.Contains(output, "Error:") || !strings.Contains(output, "not found") {
		t.Errorf("executeInsertAfter() output = %v, should contain error about pattern not found", output)
	}

	// ファイルが変更されていないことを確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(gotContent) != content {
		t.Error("executeInsertAfter() should not modify file when pattern not found")
	}
}

func TestExecuteInsertAfter_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	// 同じパターンが複数回出現
	content := "Line 1\nDuplicate\nLine 2\nDuplicate\nLine 3"

	// テストファイル作成
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "Duplicate"
	insertContent := "Inserted"

	output, backupPath, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	// バックアップは作成されない（エラー）
	if backupPath != "" {
		t.Errorf("executeInsertAfter() backupPath = %v, want empty for multiple matches", backupPath)
	}

	// エラーメッセージを確認
	if !strings.Contains(output, "Error:") || !strings.Contains(output, "matched") {
		t.Errorf("executeInsertAfter() output = %v, should contain error about multiple matches", output)
	}

	// ファイルが変更されていないことを確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(gotContent) != content {
		t.Error("executeInsertAfter() should not modify file when multiple matches")
	}
}

func TestExecuteInsertAfter_InsertAtEnd(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3"

	// テストファイル作成
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 最後の行の後に挿入
	pattern := "Line 3"
	insertContent := "Inserted at end"

	output, backupPath, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	if backupPath == "" {
		t.Error("executeInsertAfter() should create backup")
	}

	// ファイル内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := "Line 1\nLine 2\nLine 3\nInserted at end"
	if string(gotContent) != expectedContent {
		t.Errorf("executeInsertAfter() file content = %q, want %q", string(gotContent), expectedContent)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Inserted after line") {
		t.Errorf("executeInsertAfter() output = %v", output)
	}
}

func TestExecuteInsertAfter_MultilineInsert(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3"

	// テストファイル作成
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "Line 2"
	// 複数行挿入（改行を含む）
	insertContent := "Inserted line A\nInserted line B"

	output, _, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	// ファイル内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// 複数行が1つのエントリとして挿入される
	contentStr := string(gotContent)
	if !strings.Contains(contentStr, "Inserted line A") {
		t.Error("executeInsertAfter() did not insert multiline content")
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Inserted after line") {
		t.Errorf("executeInsertAfter() output = %v", output)
	}
}

func TestExecuteInsertAfter_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "notexist.txt")

	pattern := "Pattern"
	insertContent := "Inserted"

	output, backupPath, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeInsertAfter() backupPath = %v, want empty", backupPath)
	}

	// エラーメッセージを確認
	if !strings.Contains(output, "Error:") {
		t.Errorf("executeInsertAfter() output = %v, should contain error", output)
	}
}

func TestExecuteInsertAfter_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	// 空ファイル作成
	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "Pattern"
	insertContent := "Inserted"

	output, backupPath, err := executeInsertAfter(testFile, pattern, insertContent)
	if err != nil {
		t.Fatalf("executeInsertAfter() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeInsertAfter() backupPath = %v, want empty for empty file", backupPath)
	}

	// エラーメッセージを確認
	if !strings.Contains(output, "Error:") || !strings.Contains(output, "not found") {
		t.Errorf("executeInsertAfter() output = %v, should contain error about pattern not found", output)
	}
}
