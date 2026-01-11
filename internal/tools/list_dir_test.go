package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteListDir_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// テストファイルとディレクトリを作成
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	output := executeListDir(tmpDir)

	// ディレクトリパスが含まれることを確認
	if !strings.Contains(output, tmpDir) {
		t.Errorf("executeListDir() output should contain directory path, got %v", output)
	}

	// ファイル名が含まれることを確認
	if !strings.Contains(output, "file1.txt") {
		t.Error("executeListDir() output should contain 'file1.txt'")
	}
	if !strings.Contains(output, "file2.txt") {
		t.Error("executeListDir() output should contain 'file2.txt'")
	}
	if !strings.Contains(output, "subdir") {
		t.Error("executeListDir() output should contain 'subdir'")
	}

	// ファイルアイコン確認
	if !strings.Contains(output, "📄") {
		t.Error("executeListDir() output should contain file icon")
	}
	// ディレクトリアイコン確認
	if !strings.Contains(output, "📁") {
		t.Error("executeListDir() output should contain directory icon")
	}
}

func TestExecuteListDir_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	output := executeListDir(tmpDir)

	// ディレクトリパスが含まれることを確認
	if !strings.Contains(output, tmpDir) {
		t.Errorf("executeListDir() output should contain directory path, got %v", output)
	}

	// ディレクトリアイコンが含まれることを確認
	if !strings.Contains(output, "📂") {
		t.Error("executeListDir() output should contain directory icon")
	}
}

func TestExecuteListDir_DirectoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "notexist")

	output := executeListDir(nonExistentDir)

	// エラーメッセージを確認
	if !strings.Contains(output, "Error") {
		t.Errorf("executeListDir() output = %v, should contain 'Error'", output)
	}
}

func TestExecuteListDir_FileSizes(t *testing.T) {
	tmpDir := t.TempDir()

	// 異なるサイズのファイルを作成
	smallFile := filepath.Join(tmpDir, "small.txt")
	largeFile := filepath.Join(tmpDir, "large.txt")

	if err := os.WriteFile(smallFile, []byte("small"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(largeFile, []byte(strings.Repeat("x", 1000)), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := executeListDir(tmpDir)

	// ファイルサイズ情報が含まれることを確認
	if !strings.Contains(output, "bytes") {
		t.Error("executeListDir() output should contain file size in bytes")
	}

	// small.txtのサイズ（5 bytes）が含まれることを確認
	if !strings.Contains(output, "small.txt") {
		t.Error("executeListDir() output should contain 'small.txt'")
	}

	// large.txtのサイズ（1000 bytes）が含まれることを確認
	if !strings.Contains(output, "large.txt") {
		t.Error("executeListDir() output should contain 'large.txt'")
	}
}

func TestExecuteListDir_MixedContent(t *testing.T) {
	tmpDir := t.TempDir()

	// 複数のファイルとディレクトリを作成
	files := []string{"a.txt", "b.txt", "c.txt"}
	dirs := []string{"dir1", "dir2"}

	for _, file := range files {
		if err := os.WriteFile(filepath.Join(tmpDir, file), []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("Failed to create test directory %s: %v", dir, err)
		}
	}

	output := executeListDir(tmpDir)

	// すべてのファイルが含まれることを確認
	for _, file := range files {
		if !strings.Contains(output, file) {
			t.Errorf("executeListDir() output should contain file '%s'", file)
		}
	}

	// すべてのディレクトリが含まれることを確認
	for _, dir := range dirs {
		if !strings.Contains(output, dir) {
			t.Errorf("executeListDir() output should contain directory '%s'", dir)
		}
	}
}

func TestExecuteListDir_SpecialCharactersInNames(t *testing.T) {
	tmpDir := t.TempDir()

	// 特殊文字を含むファイル名
	specialFile := filepath.Join(tmpDir, "file-with_special.chars.txt")
	if err := os.WriteFile(specialFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := executeListDir(tmpDir)

	// 特殊文字を含むファイル名が含まれることを確認
	if !strings.Contains(output, "file-with_special.chars.txt") {
		t.Error("executeListDir() output should contain file with special characters")
	}
}

func TestExecuteListDir_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// 隠しファイル（ドットで始まる）を作成
	hiddenFile := filepath.Join(tmpDir, ".hidden")
	if err := os.WriteFile(hiddenFile, []byte("hidden content"), 0644); err != nil {
		t.Fatalf("Failed to create hidden file: %v", err)
	}

	// 通常のファイルも作成
	normalFile := filepath.Join(tmpDir, "normal.txt")
	if err := os.WriteFile(normalFile, []byte("normal content"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	output := executeListDir(tmpDir)

	// 隠しファイルが含まれることを確認
	if !strings.Contains(output, ".hidden") {
		t.Error("executeListDir() output should contain hidden file")
	}

	// 通常のファイルも含まれることを確認
	if !strings.Contains(output, "normal.txt") {
		t.Error("executeListDir() output should contain normal file")
	}
}

func TestExecuteListDir_RelativePath(t *testing.T) {
	// カレントディレクトリのリスト取得
	output := executeListDir(".")

	// 絶対パスに変換されていることを確認
	absPath, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if !strings.Contains(output, absPath) {
		t.Errorf("executeListDir() output should contain absolute path, got %v", output)
	}
}
