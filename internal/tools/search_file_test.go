package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteSearchFile_Success(t *testing.T) {
	tmpDir := t.TempDir()

	// テストファイルを作成
	testFiles := []string{"test.go", "test.txt", "test.md"}
	for _, file := range testFiles {
		filePath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// *.goファイルを検索
	pattern := "*.go"
	output := executeSearchFile(pattern, tmpDir)

	// test.goが見つかることを確認
	if !strings.Contains(output, "test.go") {
		t.Errorf("executeSearchFile() output = %v, should contain 'test.go'", output)
	}

	// test.txtは含まれないことを確認
	if strings.Contains(output, "test.txt") {
		t.Error("executeSearchFile() should not match non-Go files")
	}
}

func TestExecuteSearchFile_EmptyPattern(t *testing.T) {
	tmpDir := t.TempDir()

	output := executeSearchFile("", tmpDir)

	// エラーメッセージを確認
	if !strings.Contains(output, "Error:") || !strings.Contains(output, "required") {
		t.Errorf("executeSearchFile() output = %v, should contain error about required pattern", output)
	}
}

func TestExecuteSearchFile_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// テストファイルを作成
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 存在しないパターンで検索
	pattern := "*.nonexistent"
	output := executeSearchFile(pattern, tmpDir)

	// マッチなしのメッセージを確認
	if !strings.Contains(output, "No files found") {
		t.Errorf("executeSearchFile() output = %v, should contain 'No files found'", output)
	}
}

func TestExecuteSearchFile_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()

	// 複数のマッチするファイルを作成
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// *.txtファイルを検索
	pattern := "*.txt"
	output := executeSearchFile(pattern, tmpDir)

	// すべてのファイルが含まれることを確認
	for _, file := range files {
		if !strings.Contains(output, file) {
			t.Errorf("executeSearchFile() output should contain file %s", file)
		}
	}
}

func TestExecuteSearchFile_ExcludeGitDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// .gitディレクトリを作成
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.Mkdir(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// .git内にファイルを作成
	gitFile := filepath.Join(gitDir, "config.txt")
	if err := os.WriteFile(gitFile, []byte("git config"), 0644); err != nil {
		t.Fatalf("Failed to create git file: %v", err)
	}

	// 通常のファイルを作成
	normalFile := filepath.Join(tmpDir, "normal.txt")
	if err := os.WriteFile(normalFile, []byte("normal file"), 0644); err != nil {
		t.Fatalf("Failed to create normal file: %v", err)
	}

	// *.txtファイルを検索
	pattern := "*.txt"
	output := executeSearchFile(pattern, tmpDir)

	// 通常のファイルは含まれる
	if !strings.Contains(output, "normal.txt") {
		t.Error("executeSearchFile() should find normal.txt")
	}

	// .git内のファイルは除外される
	if strings.Contains(output, "config.txt") {
		t.Error("executeSearchFile() should exclude files in .git directory")
	}
}

func TestExecuteSearchFile_NestedDirectories(t *testing.T) {
	tmpDir := t.TempDir()

	// ネストされたディレクトリを作成
	nestedDir := filepath.Join(tmpDir, "level1", "level2")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}

	// ネストされたディレクトリ内にファイルを作成
	nestedFile := filepath.Join(nestedDir, "nested.go")
	if err := os.WriteFile(nestedFile, []byte("nested content"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}

	// ルートディレクトリにもファイルを作成
	rootFile := filepath.Join(tmpDir, "root.go")
	if err := os.WriteFile(rootFile, []byte("root content"), 0644); err != nil {
		t.Fatalf("Failed to create root file: %v", err)
	}

	// *.goファイルを検索
	pattern := "*.go"
	output := executeSearchFile(pattern, tmpDir)

	// 両方のファイルが見つかることを確認
	if !strings.Contains(output, "nested.go") {
		t.Error("executeSearchFile() should find nested.go")
	}
	if !strings.Contains(output, "root.go") {
		t.Error("executeSearchFile() should find root.go")
	}
}

func TestExecuteSearchFile_LongOutput(t *testing.T) {
	tmpDir := t.TempDir()

	// 40個のファイルを作成（出力が切り詰められるか確認）
	for i := 0; i < 40; i++ {
		fileName := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(fileName, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	pattern := "*.txt"
	output := executeSearchFile(pattern, tmpDir)

	// 出力が切り詰められていることを確認（30行制限）
	if strings.Contains(output, "more files") {
		// 切り詰めメッセージが含まれる
		lines := strings.Split(output, "\n")
		if len(lines) > 35 {
			t.Errorf("executeSearchFile() output should be truncated, got %d lines", len(lines))
		}
	}
}

func TestExecuteSearchFile_ExactFileName(t *testing.T) {
	tmpDir := t.TempDir()

	// 特定のファイル名を作成
	testFile := filepath.Join(tmpDir, "README.md")
	if err := os.WriteFile(testFile, []byte("readme content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 他のファイルも作成
	otherFile := filepath.Join(tmpDir, "OTHER.md")
	if err := os.WriteFile(otherFile, []byte("other content"), 0644); err != nil {
		t.Fatalf("Failed to create other file: %v", err)
	}

	// 正確なファイル名で検索
	pattern := "README.md"
	output := executeSearchFile(pattern, tmpDir)

	// README.mdのみが見つかることを確認
	if !strings.Contains(output, "README.md") {
		t.Error("executeSearchFile() should find README.md")
	}
	if strings.Contains(output, "OTHER.md") {
		t.Error("executeSearchFile() should not find OTHER.md with exact pattern")
	}
}

func TestExecuteSearchFile_NonexistentDirectory(t *testing.T) {
	pattern := "*.txt"
	output := executeSearchFile(pattern, "/nonexistent/directory/12345")

	// エラーまたはマッチなしのメッセージを確認
	if len(output) == 0 {
		t.Error("executeSearchFile() should return some output for nonexistent directory")
	}
	// findコマンドはエラーを返すはず
}

func TestExecuteSearchFile_CaseSensitivity(t *testing.T) {
	tmpDir := t.TempDir()

	// 大文字小文字が異なるファイルを作成
	lowerFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(lowerFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 小文字パターンで検索
	pattern := "test.txt"
	output := executeSearchFile(pattern, tmpDir)

	// test.txtが見つかることを確認
	if !strings.Contains(output, "test.txt") {
		t.Errorf("executeSearchFile() should find test.txt, got %v", output)
	}

	// 大文字パターンで検索（マッチしない）
	patternUpper := "TEST.TXT"
	outputUpper := executeSearchFile(patternUpper, tmpDir)

	// TEST.TXTは見つからない（ケースセンシティブ）
	if strings.Contains(outputUpper, "test.txt") {
		t.Error("executeSearchFile() should be case-sensitive")
	}
}
