package agent

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestToolCache_FileCache(t *testing.T) {
	cache := NewToolCache()

	// テスト用一時ファイル作成
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// キャッシュミス
	_, hit := cache.GetFile(testFile)
	if hit {
		t.Error("Expected cache miss, got hit")
	}

	// キャッシュに保存
	cache.SetFile(testFile, content)

	// キャッシュヒット
	cached, hit := cache.GetFile(testFile)
	if !hit {
		t.Error("Expected cache hit, got miss")
	}
	if cached != content {
		t.Errorf("Expected %q, got %q", content, cached)
	}

	// ファイル変更後はキャッシュ無効
	time.Sleep(10 * time.Millisecond) // mtimeの変化を確実に
	newContent := "Updated content"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}

	_, hit = cache.GetFile(testFile)
	if hit {
		t.Error("Expected cache miss after file modification")
	}
}

func TestToolCache_DirCache(t *testing.T) {
	cache := NewToolCache()

	// テスト用一時ディレクトリ
	tmpDir := t.TempDir()
	result := "📂 " + tmpDir

	// キャッシュミス
	_, hit := cache.GetDir(tmpDir)
	if hit {
		t.Error("Expected cache miss, got hit")
	}

	// キャッシュに保存
	cache.SetDir(tmpDir, result)

	// キャッシュヒット
	cached, hit := cache.GetDir(tmpDir)
	if !hit {
		t.Error("Expected cache hit, got miss")
	}
	if cached != result {
		t.Errorf("Expected %q, got %q", result, cached)
	}

	// ディレクトリ変更後はキャッシュ無効（ファイル追加）
	time.Sleep(10 * time.Millisecond)
	newFile := filepath.Join(tmpDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	_, hit = cache.GetDir(tmpDir)
	if hit {
		t.Error("Expected cache miss after directory modification")
	}
}

func TestToolCache_InvalidateFile(t *testing.T) {
	cache := NewToolCache()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// キャッシュに保存
	cache.SetFile(testFile, "test")

	// キャッシュヒット確認
	_, hit := cache.GetFile(testFile)
	if !hit {
		t.Error("Expected cache hit")
	}

	// 無効化
	cache.InvalidateFile(testFile)

	// キャッシュミス確認
	_, hit = cache.GetFile(testFile)
	if hit {
		t.Error("Expected cache miss after invalidation")
	}
}

func TestToolCache_Clear(t *testing.T) {
	cache := NewToolCache()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// キャッシュに保存
	cache.SetFile(testFile, "test")
	cache.SetDir(tmpDir, "dir result")
	cache.SetSearch("pattern", ".", "search result")

	// 全クリア
	cache.Clear()

	// 全てキャッシュミス
	_, hit := cache.GetFile(testFile)
	if hit {
		t.Error("Expected file cache miss after clear")
	}
	_, hit = cache.GetDir(tmpDir)
	if hit {
		t.Error("Expected dir cache miss after clear")
	}
	_, hit = cache.GetSearch("pattern", ".")
	if hit {
		t.Error("Expected search cache miss after clear")
	}
}

func TestToolCache_LargeFileNotCached(t *testing.T) {
	cache := NewToolCache()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")

	// 1MB以上のファイル
	largeContent := make([]byte, MaxFileCacheSize+1)
	if err := os.WriteFile(testFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	// キャッシュに保存しようとしても無視される
	cache.SetFile(testFile, string(largeContent))

	// キャッシュミス（保存されていない）
	_, hit := cache.GetFile(testFile)
	if hit {
		t.Error("Large file should not be cached")
	}
}

func TestToolCache_Stats(t *testing.T) {
	cache := NewToolCache()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 初期状態
	files, dirs, searches := cache.Stats()
	if files != 0 || dirs != 0 || searches != 0 {
		t.Errorf("Expected (0, 0, 0), got (%d, %d, %d)", files, dirs, searches)
	}

	// キャッシュ追加後
	cache.SetFile(testFile, "test")
	cache.SetDir(tmpDir, "dir")
	cache.SetSearch("pattern", ".", "search result")

	files, dirs, searches = cache.Stats()
	if files != 1 || dirs != 1 || searches != 1 {
		t.Errorf("Expected (1, 1, 1), got (%d, %d, %d)", files, dirs, searches)
	}
}

func TestToolCache_SearchCache(t *testing.T) {
	cache := NewToolCache()

	pattern := "func main"
	path := "."
	result := "main.go:10: func main() {"

	// キャッシュミス
	_, hit := cache.GetSearch(pattern, path)
	if hit {
		t.Error("Expected cache miss, got hit")
	}

	// キャッシュに保存
	cache.SetSearch(pattern, path, result)

	// キャッシュヒット
	cached, hit := cache.GetSearch(pattern, path)
	if !hit {
		t.Error("Expected cache hit, got miss")
	}
	if cached != result {
		t.Errorf("Expected %q, got %q", result, cached)
	}

	// 異なるパターンはキャッシュミス
	_, hit = cache.GetSearch("different pattern", path)
	if hit {
		t.Error("Expected cache miss for different pattern")
	}

	// 異なるパスはキャッシュミス
	_, hit = cache.GetSearch(pattern, "/different/path")
	if hit {
		t.Error("Expected cache miss for different path")
	}
}

func TestToolCache_ClearSearchCache(t *testing.T) {
	cache := NewToolCache()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 各種キャッシュを保存
	cache.SetFile(testFile, "file content")
	cache.SetDir(tmpDir, "dir result")
	cache.SetSearch("pattern1", ".", "search result 1")
	cache.SetSearch("pattern2", "/path", "search result 2")

	// 検索キャッシュだけクリア
	cache.ClearSearchCache()

	// ファイルキャッシュはヒット
	_, hit := cache.GetFile(testFile)
	if !hit {
		t.Error("Expected file cache hit after ClearSearchCache")
	}

	// ディレクトリキャッシュはヒット
	_, hit = cache.GetDir(tmpDir)
	if !hit {
		t.Error("Expected dir cache hit after ClearSearchCache")
	}

	// 検索キャッシュはミス
	_, hit = cache.GetSearch("pattern1", ".")
	if hit {
		t.Error("Expected search cache miss after ClearSearchCache")
	}
	_, hit = cache.GetSearch("pattern2", "/path")
	if hit {
		t.Error("Expected search cache miss after ClearSearchCache")
	}
}
