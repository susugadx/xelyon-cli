package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockValidatePathForTest はテスト用にValidatePathをモック化
func mockValidatePathForTest(t *testing.T) {
	oldValidatePath := ValidatePath
	ValidatePath = func(path string) (string, error) {
		return filepath.Abs(path)
	}
	t.Cleanup(func() { ValidatePath = oldValidatePath })
}

func TestExecuteAppendFile_NewFile(t *testing.T) {
	mockValidatePathForTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	content := "Hello, world!\nSecond line."

	output, backupPath, err := executeAppendFile(testFile, content)
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	// バックアップは作成されない（新規ファイル）
	if backupPath != "" {
		t.Errorf("executeAppendFile() backupPath = %v, want empty for new file", backupPath)
	}

	// ファイルが作成されたことを確認
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("executeAppendFile() did not create file")
	}

	// 内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(gotContent) != content {
		t.Errorf("executeAppendFile() file content = %v, want %v", string(gotContent), content)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Successfully appended") {
		t.Errorf("executeAppendFile() output = %v, should contain 'Successfully appended'", output)
	}
}

func TestExecuteAppendFile_ExistingFile(t *testing.T) {
	mockValidatePathForTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	originalContent := "Original line 1\nOriginal line 2\n"
	appendContent := "Appended line 1\nAppended line 2"

	// 既存ファイル作成
	if err := os.WriteFile(testFile, []byte(originalContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output, backupPath, err := executeAppendFile(testFile, appendContent)
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	// バックアップが作成されたことを確認
	if backupPath == "" {
		t.Error("executeAppendFile() should create backup for existing file")
	}

	// ファイル内容を確認（元の内容 + 追加内容）
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := originalContent + appendContent
	if string(gotContent) != expectedContent {
		t.Errorf("executeAppendFile() file content = %v, want %v", string(gotContent), expectedContent)
	}

	// 出力メッセージを確認
	if !strings.Contains(output, "Successfully appended") {
		t.Errorf("executeAppendFile() output = %v, should contain 'Successfully appended'", output)
	}
}

func TestExecuteAppendFile_EmptyPath(t *testing.T) {
	output, backupPath, err := executeAppendFile("", "content")
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeAppendFile() backupPath = %v, want empty", backupPath)
	}

	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("executeAppendFile() output = %v, should contain 'path is empty'", output)
	}
}

func TestExecuteAppendFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	output, backupPath, err := executeAppendFile(testFile, "")
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeAppendFile() backupPath = %v, want empty", backupPath)
	}

	if !strings.Contains(output, "Error: content is empty") {
		t.Errorf("executeAppendFile() output = %v, should contain 'content is empty'", output)
	}

	// ファイルが作成されていないことを確認
	if _, err := os.Stat(testFile); err == nil {
		t.Error("executeAppendFile() should not create file with empty content")
	}
}

func TestExecuteAppendFile_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	maliciousPath := filepath.Join(tmpDir, "../../../etc/passwd")
	content := "malicious content"

	output, backupPath, err := executeAppendFile(maliciousPath, content)
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	if backupPath != "" {
		t.Errorf("executeAppendFile() backupPath = %v, want empty for invalid path", backupPath)
	}

	if !strings.Contains(output, "Error:") {
		t.Errorf("executeAppendFile() output = %v, should contain 'Error:'", output)
	}
}

func TestExecuteAppendFile_MultipleAppends(t *testing.T) {
	mockValidatePathForTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// 1回目の追記
	content1 := "First append\n"
	output1, _, err := executeAppendFile(testFile, content1)
	if err != nil {
		t.Fatalf("executeAppendFile() 1st error = %v", err)
	}
	if !strings.Contains(output1, "Successfully appended") {
		t.Errorf("executeAppendFile() 1st output = %v", output1)
	}

	// 2回目の追記
	content2 := "Second append\n"
	_, backupPath, err := executeAppendFile(testFile, content2)
	if err != nil {
		t.Fatalf("executeAppendFile() 2nd error = %v", err)
	}
	if backupPath == "" {
		t.Error("executeAppendFile() 2nd should create backup")
	}

	// 3回目の追記
	content3 := "Third append"
	output3, _, err := executeAppendFile(testFile, content3)
	if err != nil {
		t.Fatalf("executeAppendFile() 3rd error = %v", err)
	}
	if !strings.Contains(output3, "Successfully appended") {
		t.Errorf("executeAppendFile() 3rd output = %v", output3)
	}

	// 最終内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := content1 + content2 + content3
	if string(gotContent) != expectedContent {
		t.Errorf("executeAppendFile() final content = %v, want %v", string(gotContent), expectedContent)
	}
}

func TestExecuteAppendFile_LargeContent(t *testing.T) {
	mockValidatePathForTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 1000行のコンテンツ
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("Line ")
		sb.WriteString(string(rune('0' + (i % 10))))
		sb.WriteString("\n")
	}
	largeContent := sb.String()

	output, _, err := executeAppendFile(testFile, largeContent)
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	if !strings.Contains(output, "Successfully appended") {
		t.Errorf("executeAppendFile() output = %v", output)
	}

	// ファイルサイズを確認
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	if info.Size() != int64(len(largeContent)) {
		t.Errorf("executeAppendFile() file size = %d, want %d", info.Size(), len(largeContent))
	}
}

func TestExecuteAppendFile_SpecialCharacters(t *testing.T) {
	mockValidatePathForTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "special.txt")
	specialContent := "Hello 世界 🌍\n日本語テスト\n特殊文字: @#$%^&*()"

	output, _, err := executeAppendFile(testFile, specialContent)
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	if !strings.Contains(output, "Successfully appended") {
		t.Errorf("executeAppendFile() output = %v", output)
	}

	// 内容を確認（UTF-8保持）
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(gotContent) != specialContent {
		t.Error("executeAppendFile() did not preserve special characters")
	}

	if !strings.Contains(string(gotContent), "世界") {
		t.Error("executeAppendFile() did not preserve Japanese characters")
	}
	if !strings.Contains(string(gotContent), "🌍") {
		t.Error("executeAppendFile() did not preserve emoji")
	}
}

func TestExecuteAppendFile_NoNewlineAtEnd(t *testing.T) {
	mockValidatePathForTest(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// 末尾に改行なしのファイル
	if err := os.WriteFile(testFile, []byte("Line without newline"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 追記
	appendContent := "\nNew line after"
	output, _, err := executeAppendFile(testFile, appendContent)
	if err != nil {
		t.Fatalf("executeAppendFile() error = %v", err)
	}

	if !strings.Contains(output, "Successfully appended") {
		t.Errorf("executeAppendFile() output = %v", output)
	}

	// 内容を確認
	gotContent, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	expectedContent := "Line without newline\nNew line after"
	if string(gotContent) != expectedContent {
		t.Errorf("executeAppendFile() content = %v, want %v", string(gotContent), expectedContent)
	}
}
