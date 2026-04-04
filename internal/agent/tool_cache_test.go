package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
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
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}
	futureModTime := info.ModTime().Add(time.Second)

	// キャッシュヒット
	cached, hit := cache.GetFile(testFile)
	if !hit {
		t.Error("Expected cache hit, got miss")
	}
	if cached != content {
		t.Errorf("Expected %q, got %q", content, cached)
	}

	// ファイル変更後はキャッシュ無効
	newContent := "Updated content"
	if err := os.WriteFile(testFile, []byte(newContent), 0644); err != nil {
		t.Fatalf("Failed to update test file: %v", err)
	}
	if err := os.Chtimes(testFile, futureModTime, futureModTime); err != nil {
		t.Fatalf("Failed to update file modtime: %v", err)
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
	info, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Failed to stat temp dir: %v", err)
	}
	futureModTime := info.ModTime().Add(time.Second)

	// キャッシュヒット
	cached, hit := cache.GetDir(tmpDir)
	if !hit {
		t.Error("Expected cache hit, got miss")
	}
	if cached != result {
		t.Errorf("Expected %q, got %q", result, cached)
	}

	// ディレクトリ変更後はキャッシュ無効（ファイル追加）
	newFile := filepath.Join(tmpDir, "new.txt")
	if err := os.WriteFile(newFile, []byte("new"), 0644); err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}
	if err := os.Chtimes(tmpDir, futureModTime, futureModTime); err != nil {
		t.Fatalf("Failed to update dir modtime: %v", err)
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
	cache.SetSearch("pattern", ".", "search result", nil)

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
	cache.SetSearch("pattern", ".", "search result", nil)

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
	cache.SetSearch(pattern, path, result, nil)

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
	cache.SetSearch("pattern1", ".", "search result 1", nil)
	cache.SetSearch("pattern2", "/path", "search result 2", nil)

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

func TestToolCache_InvalidateSearchCacheForFile(t *testing.T) {
	c := NewToolCache()
	// absPathを含む検索結果をキャッシュ
	c.SetSearch("pattern1", "/project", "result1", []string{"/project/main.go"})
	c.SetSearch("pattern2", "/project", "result2", []string{"/project/utils.go"})
	c.SetSearch("pattern3", "/project", "result3", []string{"/project/main.go"})

	// main.go だけ無効化
	c.InvalidateSearchCacheForFile("/project/main.go")

	// main.go を含むキャッシュは消えている
	_, ok1 := c.GetSearch("pattern1", "/project")
	if ok1 {
		t.Error("expected cache miss for pattern1 (contains main.go)")
	}
	_, ok3 := c.GetSearch("pattern3", "/project")
	if ok3 {
		t.Error("expected cache miss for pattern3 (contains main.go)")
	}

	// utils.go のみのキャッシュは残っている
	_, ok2 := c.GetSearch("pattern2", "/project")
	if !ok2 {
		t.Error("expected cache hit for pattern2 (only contains utils.go)")
	}
}

func TestToolCache_InvalidateSearchCacheForFile_NotifiesOnlyWhenDeleted(t *testing.T) {
	tools.RegisterSearchCacheLifecycleHooks(nil, nil, nil)
	t.Cleanup(func() {
		tools.RegisterSearchCacheLifecycleHooks(nil, nil, nil)
	})

	invalidateCount := 0
	var invalidatedKeys []string
	tools.RegisterSearchCacheLifecycleHooks(nil, func(keys []string) {
		invalidateCount++
		invalidatedKeys = append([]string(nil), keys...)
	}, nil)

	c := NewToolCache()
	c.SetSearch("pattern-main", "/project", "result-main", []string{"/project/main.go"})

	// unrelated path: no entry deleted -> no notification
	c.InvalidateSearchCacheForFile("/project/unrelated.go")
	if invalidateCount != 0 {
		t.Fatalf("expected no invalidate hook calls for unrelated path, got %d", invalidateCount)
	}

	// related path: entry deleted -> notification should fire once
	c.InvalidateSearchCacheForFile("/project/main.go")
	if invalidateCount != 1 {
		t.Fatalf("expected one invalidate hook call for related path, got %d", invalidateCount)
	}
	if len(invalidatedKeys) != 1 || invalidatedKeys[0] != searchCacheKey("pattern-main", "/project") {
		t.Fatalf("unexpected invalidated keys: %v", invalidatedKeys)
	}
}

func TestToolCache_SearchEvictionTriggersBundleCacheHook(t *testing.T) {
	tools.RegisterSearchCacheLifecycleHooks(nil, nil, nil)
	t.Cleanup(func() {
		tools.RegisterSearchCacheLifecycleHooks(nil, nil, nil)
	})

	evicted := 0
	var evictedKeys []string
	tools.RegisterSearchCacheLifecycleHooks(nil, nil, func(keys []string) {
		evicted++
		evictedKeys = append(evictedKeys, keys...)
	})

	cache := NewToolCache()
	for i := 0; i < MaxSearchCacheEntries+1; i++ {
		cache.SetSearch(fmt.Sprintf("pattern-%d", i), "/project", "result", nil)
	}

	if evicted == 0 {
		t.Fatal("expected eviction hook to be triggered")
	}
	if len(evictedKeys) == 0 {
		t.Fatal("expected evicted keys to be reported")
	}
}

func TestToolCache_RecentSearchAffectedFilesAfterSymbolSearch(t *testing.T) {
	cache := NewToolCache()
	dir := t.TempDir()
	file := filepath.Join(dir, "run.go")
	if err := os.WriteFile(file, []byte("package example\n\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	result := search.ExecuteSearchCodeWithCache(cache, search.SearchOptions{
		Pattern: "Run",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run() {}", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "agent-search-affected",
	})
	if !strings.Contains(result, "Run") {
		t.Fatalf("expected symbol search result, got:\n%s", result)
	}

	affected := cache.RecentSearchAffectedFiles(5)
	if len(affected) == 0 {
		t.Fatal("expected affected files after symbol search")
	}
	if affected[0] != file {
		t.Fatalf("expected first affected file %s, got %v", file, affected)
	}
}

func TestToolCache_FileCacheLRU(t *testing.T) {
	cache := NewToolCache()
	tmpDir := t.TempDir()

	for i := 0; i < MaxFileCacheEntries+1; i++ {
		p := filepath.Join(tmpDir, fmt.Sprintf("f%03d.txt", i))
		if err := os.WriteFile(p, []byte("v"), 0644); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}
		cache.SetFile(p, "v")
	}

	files, _, _ := cache.Stats()
	if files > MaxFileCacheEntries {
		t.Fatalf("file cache should be capped at %d, got %d", MaxFileCacheEntries, files)
	}
}

func TestIsNegativeResult(t *testing.T) {
	tests := []struct {
		content string
		want    bool
	}{
		{"No matches found", true},
		{"  No matches found\n", true},
		{"Error: pattern not found", true},
		{"Error: old_str not found in file", true},
		{"Error reading file: /foo", true},
		{"Error: something", true},
		{"package main", false},
		{"result contains Error: in middle", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isNegativeResult(tt.content)
		if got != tt.want {
			t.Errorf("isNegativeResult(%q) = %v, want %v", tt.content, got, tt.want)
		}
	}
}

func TestNegativeCacheKey(t *testing.T) {
	args := map[string]interface{}{"path": "/foo.go", "pattern": "func main"}
	key1 := negativeCacheKey("search_code", args)
	key2 := negativeCacheKey("search_code", args)
	if key1 != key2 {
		t.Errorf("same args should produce same key: %q != %q", key1, key2)
	}

	// 異なるツール名→異なるキー
	key3 := negativeCacheKey("read_file", args)
	if key1 == key3 {
		t.Error("different tool name should produce different key")
	}

	// 異なる引数→異なるキー
	args2 := map[string]interface{}{"path": "/bar.go", "pattern": "func main"}
	key4 := negativeCacheKey("search_code", args2)
	if key1 == key4 {
		t.Error("different args should produce different key")
	}
}

func TestToolCache_NegativeCache_SetAndCheck(t *testing.T) {
	cache := NewToolCache()
	args := map[string]interface{}{"pattern": "nonexistent", "path": "."}

	// 初回はミス
	_, hit := cache.CheckNegativeCache("search_code", args)
	if hit {
		t.Error("expected negative cache miss on first check")
	}

	// エラー結果をセット
	cache.SetNegativeCache("search_code", args, "No matches found")

	// ヒット
	result, hit := cache.CheckNegativeCache("search_code", args)
	if !hit {
		t.Error("expected negative cache hit")
	}
	if result != "No matches found" {
		t.Errorf("expected 'No matches found', got %q", result)
	}
}

func TestToolCache_NegativeCache_MultilineError(t *testing.T) {
	cache := NewToolCache()
	args := map[string]interface{}{"path": "/foo.go"}

	// 複数行エラーは1行目だけ保存される
	cache.SetNegativeCache("read_file", args, "Error: file not found\nstack trace line 1\nstack trace line 2")

	result, hit := cache.CheckNegativeCache("read_file", args)
	if !hit {
		t.Error("expected negative cache hit")
	}
	if result != "Error: file not found" {
		t.Errorf("expected first line only, got %q", result)
	}
}

func TestToolCache_NegativeCache_IgnoresNonError(t *testing.T) {
	cache := NewToolCache()
	args := map[string]interface{}{"path": "/foo.go"}

	// 正常な結果はネガティブキャッシュに保存されない
	cache.SetNegativeCache("read_file", args, "package main\n\nfunc main() {}")

	_, hit := cache.CheckNegativeCache("read_file", args)
	if hit {
		t.Error("non-error result should not be cached in negative cache")
	}
}

func TestToolCache_NegativeCache_TTLExpiry(t *testing.T) {
	cache := NewToolCache()
	args := map[string]interface{}{"pattern": "x"}

	cache.SetNegativeCache("search_code", args, "No matches found")

	// 手動で CachedAt を過去にセット
	key := negativeCacheKey("search_code", args)
	cache.mu.Lock()
	entry := cache.negatives[key]
	entry.CachedAt = time.Now().Add(-NegativeCacheTTL - time.Second)
	cache.negatives[key] = entry
	cache.mu.Unlock()

	// TTL 切れでミス
	_, hit := cache.CheckNegativeCache("search_code", args)
	if hit {
		t.Error("expected negative cache miss after TTL expiry")
	}

	// 期限切れエントリは削除されている
	cache.mu.RLock()
	_, exists := cache.negatives[key]
	cache.mu.RUnlock()
	if exists {
		t.Error("expired entry should be deleted")
	}
}

func TestToolCache_NegativeCache_ClearResetsAll(t *testing.T) {
	cache := NewToolCache()
	args := map[string]interface{}{"pattern": "x"}

	cache.SetNegativeCache("search_code", args, "No matches found")
	cache.Clear()

	_, hit := cache.CheckNegativeCache("search_code", args)
	if hit {
		t.Error("expected negative cache miss after Clear")
	}
}

// --- Save/Load 永続化テスト ---

func TestToolCache_SaveLoad_RoundTrip(t *testing.T) {
	// テスト用ディレクトリでSave/Loadを実行
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// テスト用ファイル作成
	testFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(testFile, []byte("package main"), 0644); err != nil {
		t.Fatal(err)
	}
	testDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(testDir, 0755); err != nil {
		t.Fatal(err)
	}

	// キャッシュにデータを設定
	cache1 := NewToolCache()
	cache1.SetFile(testFile, "package main")
	cache1.SetDir(testDir, "file1.go\nfile2.go")
	cache1.SetSearch("func", ".", "results", []string{testFile})

	// Save
	if err := cache1.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// 新しいキャッシュでLoad
	cache2 := NewToolCache()
	if err := cache2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// ファイルキャッシュが復元されていること
	if content, hit := cache2.GetFile(testFile); !hit {
		t.Error("expected file cache hit after Load")
	} else if content != "package main" {
		t.Errorf("expected 'package main', got %q", content)
	}

	// ディレクトリキャッシュが復元されていること
	if content, hit := cache2.GetDir(testDir); !hit {
		t.Error("expected dir cache hit after Load")
	} else if content != "file1.go\nfile2.go" {
		t.Errorf("expected dir listing, got %q", content)
	}

	// 検索キャッシュは永続化されないこと
	if _, hit := cache2.GetSearch("func", "."); hit {
		t.Error("search cache should NOT be persisted")
	}
}

func TestToolCache_Load_InvalidatesModified(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	// キャッシュ保存
	cache1 := NewToolCache()
	cache1.SetFile(testFile, "old content")
	if err := cache1.Save(); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatal(err)
	}
	futureModTime := info.ModTime().Add(time.Second)

	// ファイルを変更
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(testFile, futureModTime, futureModTime); err != nil {
		t.Fatal(err)
	}

	// Load → 変更されたファイルは復元されないこと
	cache2 := NewToolCache()
	if err := cache2.Load(); err != nil {
		t.Fatal(err)
	}

	if _, hit := cache2.GetFile(testFile); hit {
		t.Error("modified file should NOT be restored from cache")
	}
}

func TestToolCache_Load_InvalidatesDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	cache1 := NewToolCache()
	cache1.SetFile(testFile, "content")
	if err := cache1.Save(); err != nil {
		t.Fatal(err)
	}

	// ファイル削除
	os.Remove(testFile)

	cache2 := NewToolCache()
	if err := cache2.Load(); err != nil {
		t.Fatal(err)
	}

	if _, hit := cache2.GetFile(testFile); hit {
		t.Error("deleted file should NOT be restored from cache")
	}
}

func TestToolCache_Save_EmptyCacheRemovesFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// まずキャッシュファイルを作成
	cacheDir := filepath.Join(tmpDir, ".xelyon", "cache")
	os.MkdirAll(cacheDir, 0755)
	cacheFile := filepath.Join(tmpDir, ".xelyon", "cache", "tool_cache.json")
	_ = os.WriteFile(cacheFile, []byte(`{"files":{},"dirs":{}}`), 0600)

	// 空キャッシュで Save → ファイル削除
	cache := NewToolCache()
	if err := cache.Save(); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Error("expected cache file to be removed for empty cache")
	}
}

func TestToolCache_Load_CorruptedFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// 壊れたJSONを書き込み
	cacheDir := filepath.Join(tmpDir, ".xelyon", "cache")
	os.MkdirAll(cacheDir, 0755)
	cacheFile := filepath.Join(tmpDir, ".xelyon", "cache", "tool_cache.json")
	_ = os.WriteFile(cacheFile, []byte("not json{{{"), 0600)

	cache := NewToolCache()
	err := cache.Load()
	if err != nil {
		t.Errorf("Load should not return error for corrupted file, got: %v", err)
	}

	// 壊れたファイルは削除されること
	if _, statErr := os.Stat(cacheFile); !os.IsNotExist(statErr) {
		t.Error("expected corrupted cache file to be removed")
	}
}

func TestToolCache_Load_NoFile(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cache := NewToolCache()
	err := cache.Load()
	if err != nil {
		t.Errorf("Load should not return error when no cache file exists, got: %v", err)
	}
}

func TestToolCache_Save_SizeLimit(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// 大量のキャッシュエントリを作成（10MB超を狙う）
	cache := NewToolCache()
	bigContent := strings.Repeat("x", 500*1024) // 500KB per entry
	for i := 0; i < 25; i++ {
		f := filepath.Join(tmpDir, fmt.Sprintf("big_%d.txt", i))
		_ = os.WriteFile(f, []byte(bigContent), 0644)
		cache.SetFile(f, bigContent)
	}

	// Save → 10MB超ならスキップ
	if err := cache.Save(); err != nil {
		t.Fatal(err)
	}

	cacheFile := filepath.Join(tmpDir, ".xelyon", "cache", "tool_cache.json")
	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		// ファイルが存在する場合はサイズが10MB以下であること
		info, _ := os.Stat(cacheFile)
		if info != nil && info.Size() > int64(maxCacheFileSize) {
			t.Errorf("cache file should not exceed %d bytes, got %d", maxCacheFileSize, info.Size())
		}
	}
}
