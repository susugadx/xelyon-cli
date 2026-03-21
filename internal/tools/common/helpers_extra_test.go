package common

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindWithNormalizedWhitespace_BasicMatch(t *testing.T) {
	content := "func main() {\n\treturn\n}"
	pattern := "return"

	found, startIdx, endIdx := FindWithNormalizedWhitespace(content, pattern)
	if !found {
		t.Fatal("expected pattern to be found")
	}
	if startIdx < 0 || endIdx < 0 {
		t.Fatalf("expected valid indices, got start=%d end=%d", startIdx, endIdx)
	}
	// 抽出した部分文字列にパターンが含まれることを確認
	extracted := content[startIdx : endIdx+1]
	if extracted != "\treturn" {
		t.Errorf("expected extracted text to be %q, got %q", "\treturn", extracted)
	}
}

func TestFindWithNormalizedWhitespace_NotFound(t *testing.T) {
	content := "hello\nworld"
	pattern := "missing"

	found, startIdx, endIdx := FindWithNormalizedWhitespace(content, pattern)
	if found {
		t.Error("expected pattern not to be found")
	}
	if startIdx != -1 || endIdx != -1 {
		t.Errorf("expected indices to be -1, got start=%d end=%d", startIdx, endIdx)
	}
}

func TestFindWithNormalizedWhitespace_TabsVsSpaces(t *testing.T) {
	// タブ付きコンテンツ
	content := "\tfunc hello() {}"
	// スペース付きパターン（正規化後に一致するはず）
	pattern := "    func hello() {}"

	found, startIdx, endIdx := FindWithNormalizedWhitespace(content, pattern)
	if !found {
		t.Fatal("expected tabs to match spaces after normalization")
	}
	if startIdx < 0 || endIdx < 0 {
		t.Fatalf("expected valid indices, got start=%d end=%d", startIdx, endIdx)
	}
}

func TestFindWithNormalizedWhitespace_MultipleSpaces(t *testing.T) {
	// 複数スペースのインデント
	content := "        deeply indented"
	// 同じ行が正規化後に一致する
	pattern := "deeply indented"

	found, startIdx, endIdx := FindWithNormalizedWhitespace(content, pattern)
	if !found {
		t.Fatal("expected multiple leading spaces to match after normalization")
	}
	if startIdx < 0 || endIdx < 0 {
		t.Fatalf("expected valid indices, got start=%d end=%d", startIdx, endIdx)
	}
}

func TestFileExistsImpl(t *testing.T) {
	// テスト用に fileExistsImpl を直接使用
	t.Run("existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		tmpFile := filepath.Join(tmpDir, "testfile.txt")
		if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
			t.Fatalf("failed to create temp file: %v", err)
		}

		if !fileExistsImpl(tmpFile) {
			t.Error("fileExistsImpl should return true for existing file")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		if fileExistsImpl("/nonexistent/path/to/file.txt") {
			t.Error("fileExistsImpl should return false for nonexistent file")
		}
	})

	t.Run("directory exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		// ディレクトリに対しても true を返す（os.Stat はディレクトリでも成功する）
		if !fileExistsImpl(tmpDir) {
			t.Error("fileExistsImpl should return true for existing directory")
		}
	})
}
