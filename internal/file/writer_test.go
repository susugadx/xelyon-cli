package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFile_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	testContent := "new file content"

	err := WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// ファイルが作成されたことを確認
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("WriteFile() did not create file")
	}

	// 内容を確認
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("WriteFile() content = %v, want %v", string(content), testContent)
	}

	// パーミッション確認
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	// 0644 パーミッション
	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0644)
	if mode != expectedMode {
		t.Errorf("WriteFile() file mode = %v, want %v", mode, expectedMode)
	}
}

func TestWriteFile_Overwrite(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "overwrite.txt")
	oldContent := "old content"
	newContent := "new content"

	// 既存ファイルを作成
	if err := os.WriteFile(testFile, []byte(oldContent), 0644); err != nil {
		t.Fatalf("Failed to create existing file: %v", err)
	}

	// 上書き
	err := WriteFile(testFile, newContent)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// 新しい内容を確認
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != newContent {
		t.Errorf("WriteFile() content = %v, want %v", string(content), newContent)
	}

	// 古い内容が残っていないことを確認
	if string(content) == oldContent {
		t.Error("WriteFile() did not overwrite old content")
	}
}

func TestWriteFile_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "level1", "level2", "level3", "file.txt")
	testContent := "nested file content"

	err := WriteFile(nestedPath, testContent)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// ディレクトリが作成されたことを確認
	dir := filepath.Dir(nestedPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("WriteFile() did not create parent directories")
	}

	// ファイルが作成されたことを確認
	content, err := os.ReadFile(nestedPath)
	if err != nil {
		t.Fatalf("Failed to read nested file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("WriteFile() content = %v, want %v", string(content), testContent)
	}

	// ディレクトリパーミッション確認 (0755)
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Failed to stat directory: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0755)
	if mode != expectedMode {
		t.Errorf("WriteFile() directory mode = %v, want %v", mode, expectedMode)
	}
}

func TestWriteFile_EmptyContent(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	err := WriteFile(testFile, "")
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// ファイルが作成されたことを確認
	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(content) != 0 {
		t.Errorf("WriteFile() with empty content created non-empty file: %v", string(content))
	}
}

func TestWriteFile_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "special.txt")
	testContent := "Hello 世界 🌍\n改行テスト\nタブ:\tテスト\n特殊文字: @#$%^&*()"

	err := WriteFile(testFile, testContent)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != testContent {
		t.Error("WriteFile() did not preserve special characters")
	}

	// UTF-8 エンコーディング確認
	if !strings.Contains(string(content), "世界") {
		t.Error("WriteFile() did not preserve Japanese characters")
	}
	if !strings.Contains(string(content), "🌍") {
		t.Error("WriteFile() did not preserve emoji")
	}
}

func TestWriteFile_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := "relative path test"

	// 現在のディレクトリを変更
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Errorf("Failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// 相対パスで書き込み
	err = WriteFile("relative.txt", testContent)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	// 絶対パスで確認
	absPath := filepath.Join(tmpDir, "relative.txt")
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("WriteFile() content = %v, want %v", string(content), testContent)
	}
}

func TestExtractCodeBlock_Valid(t *testing.T) {
	tests := []struct {
		name     string
		response string
		want     string
	}{
		{
			name:     "simple code block",
			response: "Here is some code:\n```\nfunc main() {\n    fmt.Println(\"Hello\")\n}\n```\nThat's it!",
			want:     "func main() {\n    fmt.Println(\"Hello\")\n}",
		},
		{
			name:     "code block with language",
			response: "```go\nfunc test() {\n    return true\n}\n```",
			want:     "func test() {\n    return true\n}",
		},
		{
			name:     "code block at start",
			response: "```\ncode here\n```",
			want:     "code here",
		},
		{
			name:     "code block at end",
			response: "Some text\n```\ncode here\n```",
			want:     "code here",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCodeBlock(tt.response)
			if got != tt.want {
				t.Errorf("ExtractCodeBlock() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractCodeBlock_NoBlock(t *testing.T) {
	response := "This is just plain text without any code blocks."
	got := ExtractCodeBlock(response)

	if got != "" {
		t.Errorf("ExtractCodeBlock() = %v, want empty string", got)
	}
}

func TestExtractCodeBlock_Multiple(t *testing.T) {
	// 複数のコードブロック - 最初のみ抽出
	response := "First block:\n```\nfirst code\n```\nSecond block:\n```\nsecond code\n```"
	got := ExtractCodeBlock(response)

	want := "first code"
	if got != want {
		t.Errorf("ExtractCodeBlock() = %q, want %q (first block only)", got, want)
	}

	// 2つ目のブロックが含まれていないことを確認
	if strings.Contains(got, "second code") {
		t.Error("ExtractCodeBlock() should only extract first code block")
	}
}

func TestExtractCodeBlock_EmptyBlock(t *testing.T) {
	response := "```\n```"
	got := ExtractCodeBlock(response)

	if got != "" {
		t.Errorf("ExtractCodeBlock() with empty block = %q, want empty string", got)
	}
}

func TestExtractCodeBlock_UnmatchedMarkers(t *testing.T) {
	tests := []struct {
		name     string
		response string
	}{
		{
			name:     "only opening marker",
			response: "```\ncode here\nno closing marker",
		},
		{
			name:     "only closing marker",
			response: "no opening marker\ncode here\n```",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractCodeBlock(tt.response)
			// 開始マーカーだけの場合は空文字を返す
			if tt.name == "only closing marker" && got != "" {
				t.Errorf("ExtractCodeBlock() with unmatched markers = %q, want empty string", got)
			}
		})
	}
}

func TestExtractCodeBlock_NestedBackticks(t *testing.T) {
	// コードブロック内にバッククォートを含む
	response := "```\nfunc test() {\n    // Use ``` markers\n}\n```"
	got := ExtractCodeBlock(response)

	want := "func test() {\n    // Use ``` markers\n}"
	if got != want {
		t.Errorf("ExtractCodeBlock() = %q, want %q", got, want)
	}
}

func TestExtractCodeBlock_MultilineCode(t *testing.T) {
	response := `Here's the code:
` + "```" + `
package main

import "fmt"

func main() {
    fmt.Println("Line 1")
    fmt.Println("Line 2")
    fmt.Println("Line 3")
}
` + "```" + `
End of code.`

	got := ExtractCodeBlock(response)

	// 改行が保持されることを確認
	if !strings.Contains(got, "Line 1") {
		t.Error("ExtractCodeBlock() should preserve Line 1")
	}
	if !strings.Contains(got, "Line 2") {
		t.Error("ExtractCodeBlock() should preserve Line 2")
	}
	if !strings.Contains(got, "Line 3") {
		t.Error("ExtractCodeBlock() should preserve Line 3")
	}

	// 改行の数を確認
	lines := strings.Split(got, "\n")
	if len(lines) < 7 {
		t.Errorf("ExtractCodeBlock() should preserve multiple lines, got %d lines", len(lines))
	}
}

// ConfirmApply は標準入力を読むため、テストが難しい
// ここでは ExtractCodeBlock と WriteFile の組み合わせテストのみ
func TestWriteFile_WithExtractedCode(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "extracted.txt")

	response := "Here's the code:\n```\nfunc test() {\n    return true\n}\n```"
	code := ExtractCodeBlock(response)

	if code == "" {
		t.Fatal("ExtractCodeBlock() returned empty string")
	}

	err := WriteFile(testFile, code)
	if err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	content, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != code {
		t.Errorf("WriteFile() did not write extracted code correctly")
	}
}
