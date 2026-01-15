package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecutePrependFile_NewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	content := "Prepended line 1\nPrepended line 2"

	output, backupPath, err := executePrependFile(testFile, content)
	if err != nil {
		t.Fatalf("executePrependFile() error = %v", err)
	}

	// バックアップは作成されない（新規ファイル）
	if backupPath != "" {
		t.Errorf("executePrependFile() backupPath = %v, want empty for new file", backupPath)
	}

	// ファイルが作成されたことを確認
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("executePrependFile() did not create file")
	}

	// 内容を確認（末尾に改行が追加される）
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	expectedContent := content + "\n"
	if string(gotContent) != expectedContent {
		t.Errorf("executePrependFile() file content = %v, want %v", string(gotContent), expectedContent)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Successfully prepended") {
		t.Errorf("executePrependFile() output = %v, should contain 'Successfully prepended'", output)
	}
}

func TestExecutePrependFile_ExistingFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	existingContent := "Existing line 1\nExisting line 2\n"
	prependContent := "New line 1\nNew line 2"

	// 既存ファイル作成
	if err := os.WriteFile(testFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output, backupPath, err := executePrependFile(testFile, prependContent)
	if err != nil {
		t.Fatalf("executePrependFile() error = %v", err)
	}

	// バックアップが作成されたことを確認
	if backupPath == "" {
		t.Error("executePrependFile() should create backup for existing file")
	}

	// ファイル内容を確認（追加内容 + 改行 + 既存内容）
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := prependContent + "\n" + existingContent
	if string(gotContent) != expectedContent {
		t.Errorf("executePrependFile() file content = %q, want %q", string(gotContent), expectedContent)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Successfully prepended") {
		t.Errorf("executePrependFile() output = %v, should contain 'Successfully prepended'", output)
	}
}

func TestExecutePrependFile_EmptyPath(t *testing.T) {
	output, backupPath, err := executePrependFile("", "content")
	if err != nil {
		t.Fatalf("executePrependFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executePrependFile() backupPath = %v, want empty", backupPath)
	}

	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("executePrependFile() output = %v, should contain 'path is empty'", output)
	}
}

func TestExecutePrependFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	output, backupPath, err := executePrependFile(testFile, "")
	if err != nil {
		t.Fatalf("executePrependFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executePrependFile() backupPath = %v, want empty", backupPath)
	}

	if !strings.Contains(output, "Error: content is empty") {
		t.Errorf("executePrependFile() output = %v, should contain 'content is empty'", output)
	}

	// ファイルが作成されていないことを確認
	if _, err := os.Stat(testFile); err == nil {
		t.Error("executePrependFile() should not create file with empty content")
	}
}

func TestExecutePrependFile_ContentWithTrailingNewline(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	existingContent := "Existing"
	// 末尾に改行あり
	prependContent := "Prepended\n"

	// 既存ファイル作成
	if err := os.WriteFile(testFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, _, err := executePrependFile(testFile, prependContent)
	if err != nil {
		t.Fatalf("executePrependFile() error = %v", err)
	}

	// 内容を確認（改行が重複しないこと）
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := "Prepended\nExisting"
	if string(gotContent) != expectedContent {
		t.Errorf("executePrependFile() file content = %q, want %q", string(gotContent), expectedContent)
	}
}

func TestExecutePrependFile_MultipleOperations(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// 1回目: ファイル作成
	content1 := "First"
	_, _, err := executePrependFile(testFile, content1)
	if err != nil {
		t.Fatalf("executePrependFile() 1st error = %v", err)
	}

	// 2回目: 先頭に追加
	content2 := "Second"
	_, backupPath, err := executePrependFile(testFile, content2)
	if err != nil {
		t.Fatalf("executePrependFile() 2nd error = %v", err)
	}
	if backupPath == "" {
		t.Error("executePrependFile() 2nd should create backup")
	}

	// 3回目: さらに先頭に追加
	content3 := "Third"
	_, _, err = executePrependFile(testFile, content3)
	if err != nil {
		t.Fatalf("executePrependFile() 3rd error = %v", err)
	}

	// 最終内容を確認（逆順になる）
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := "Third\nSecond\nFirst\n"
	if string(gotContent) != expectedContent {
		t.Errorf("executePrependFile() final content = %q, want %q", string(gotContent), expectedContent)
	}
}

func TestExecutePrependFile_LargeContent(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	existingContent := "Existing content"

	// 1000行のコンテンツ
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString("Line ")
		sb.WriteString(string(rune('0' + (i % 10))))
	}
	largeContent := sb.String()

	// 既存ファイル作成
	if err := os.WriteFile(testFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output, _, err := executePrependFile(testFile, largeContent)
	if err != nil {
		t.Fatalf("executePrependFile() error = %v", err)
	}

	if !strings.Contains(output, "Successfully prepended") {
		t.Errorf("executePrependFile() output = %v", output)
	}

	// ファイル内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// 先頭部分と末尾部分を確認
	contentStr := string(gotContent)
	if !strings.HasPrefix(contentStr, "Line 0") {
		t.Error("executePrependFile() should start with prepended content")
	}
	if !strings.HasSuffix(contentStr, "Existing content") {
		t.Error("executePrependFile() should end with existing content")
	}
}

func TestExecutePrependFile_SpecialCharacters(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "special.txt")
	existingContent := "Existing"
	specialContent := "Hello 世界 🌍\n日本語テスト"

	// 既存ファイル作成
	if err := os.WriteFile(testFile, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output, _, err := executePrependFile(testFile, specialContent)
	if err != nil {
		t.Fatalf("executePrependFile() error = %v", err)
	}

	if !strings.Contains(output, "Successfully prepended") {
		t.Errorf("executePrependFile() output = %v", output)
	}

	// 内容を確認（UTF-8保持）
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	contentStr := string(gotContent)
	if !strings.Contains(contentStr, "世界") {
		t.Error("executePrependFile() did not preserve Japanese characters")
	}
	if !strings.Contains(contentStr, "🌍") {
		t.Error("executePrependFile() did not preserve emoji")
	}
	if !strings.Contains(contentStr, "Existing") {
		t.Error("executePrependFile() did not preserve existing content")
	}
}
