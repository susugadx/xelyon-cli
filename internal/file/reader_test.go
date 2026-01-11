package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := "Hello, World!"
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if content != testContent {
		t.Errorf("ReadFile() = %v, want %v", content, testContent)
	}
}

func TestReadFile_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentFile := filepath.Join(tmpDir, "does_not_exist.txt")

	_, err := ReadFile(nonExistentFile)
	if err == nil {
		t.Error("ReadFile() expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "ファイルを読めません") {
		t.Errorf("ReadFile() error message should contain 'ファイルを読めません', got: %v", err)
	}
}

func TestReadFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "empty.txt")

	if err := os.WriteFile(testFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to create empty file: %v", err)
	}

	content, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if content != "" {
		t.Errorf("ReadFile() = %v, want empty string", content)
	}
}

func TestReadFile_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 10000行のファイルを作成
	largeContent := strings.Repeat("This is line content\n", 10000)

	if err := os.WriteFile(testFile, []byte(largeContent), 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	content, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if len(content) != len(largeContent) {
		t.Errorf("ReadFile() length = %v, want %v", len(content), len(largeContent))
	}

	if content != largeContent {
		t.Error("ReadFile() content mismatch for large file")
	}
}

func TestReadFile_SpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "special.txt")

	// UTF-8、絵文字、日本語を含むコンテンツ
	testContent := "Hello 世界 🌍 こんにちは\n特殊文字: @#$%^&*()\nタブ:\t改行\n"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	content, err := ReadFile(testFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if content != testContent {
		t.Errorf("ReadFile() failed to preserve special characters")
	}

	// UTF-8 エンコーディング確認
	if !strings.Contains(content, "世界") {
		t.Error("ReadFile() should preserve Japanese characters")
	}
	if !strings.Contains(content, "🌍") {
		t.Error("ReadFile() should preserve emoji")
	}
}

func TestReadFile_RelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	testContent := "relative path test"
	testFile := filepath.Join(tmpDir, "test.txt")

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

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

	// 相対パスで読み込み
	content, err := ReadFile("test.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if content != testContent {
		t.Errorf("ReadFile() = %v, want %v", content, testContent)
	}
}

func TestReadFiles_Multiple(t *testing.T) {
	tmpDir := t.TempDir()

	// 3つのファイルを作成
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")
	file3 := filepath.Join(tmpDir, "file3.txt")

	if err := os.WriteFile(file1, []byte("Content 1"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("Content 2"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}
	if err := os.WriteFile(file3, []byte("Content 3"), 0644); err != nil {
		t.Fatalf("Failed to create file3: %v", err)
	}

	result, err := ReadFiles([]string{file1, file2, file3})
	if err != nil {
		t.Fatalf("ReadFiles() error = %v", err)
	}

	// すべてのファイルが含まれていることを確認
	if !strings.Contains(result, "Content 1") {
		t.Error("ReadFiles() should contain content from file1")
	}
	if !strings.Contains(result, "Content 2") {
		t.Error("ReadFiles() should contain content from file2")
	}
	if !strings.Contains(result, "Content 3") {
		t.Error("ReadFiles() should contain content from file3")
	}

	// フォーマット確認
	if !strings.Contains(result, "## ファイル:") {
		t.Error("ReadFiles() should contain file header")
	}
	if !strings.Contains(result, "```") {
		t.Error("ReadFiles() should contain code block markers")
	}

	// 各ファイルパスが含まれていることを確認
	if !strings.Contains(result, file1) {
		t.Errorf("ReadFiles() should contain file path %s", file1)
	}
	if !strings.Contains(result, file2) {
		t.Errorf("ReadFiles() should contain file path %s", file2)
	}
	if !strings.Contains(result, file3) {
		t.Errorf("ReadFiles() should contain file path %s", file3)
	}
}

func TestReadFiles_Empty(t *testing.T) {
	result, err := ReadFiles([]string{})
	if err != nil {
		t.Fatalf("ReadFiles() error = %v", err)
	}

	if result != "" {
		t.Errorf("ReadFiles() with empty slice = %v, want empty string", result)
	}
}

func TestReadFiles_Error(t *testing.T) {
	tmpDir := t.TempDir()

	// 存在するファイルと存在しないファイルを混在
	file1 := filepath.Join(tmpDir, "exists.txt")
	file2 := filepath.Join(tmpDir, "does_not_exist.txt")

	if err := os.WriteFile(file1, []byte("Content"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	_, err := ReadFiles([]string{file1, file2})
	if err == nil {
		t.Error("ReadFiles() expected error for non-existent file, got nil")
	}

	if !strings.Contains(err.Error(), "ファイルを読めません") {
		t.Errorf("ReadFiles() error should contain 'ファイルを読めません', got: %v", err)
	}
}

func TestReadFile_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	// ディレクトリを読み込もうとする
	_, err := ReadFile(tmpDir)
	if err == nil {
		t.Error("ReadFile() expected error when reading directory, got nil")
	}
}

func TestReadFiles_SingleFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "single.txt")
	testContent := "Single file content"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := ReadFiles([]string{testFile})
	if err != nil {
		t.Fatalf("ReadFiles() error = %v", err)
	}

	if !strings.Contains(result, testContent) {
		t.Error("ReadFiles() should contain file content")
	}

	if !strings.Contains(result, "## ファイル:") {
		t.Error("ReadFiles() should contain file header for single file")
	}
}
