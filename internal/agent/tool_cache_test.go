package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestToolCache_DeduplicateResult(t *testing.T) {
	cache := NewToolCache()

	content := "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"

	// 初回: 新規登録、空文字を返す
	ref := cache.DeduplicateResult("read_file", content, 1)
	if ref != "" {
		t.Errorf("expected empty string on first call, got %q", ref)
	}

	// 同一内容で2回目: 参照文字列を返す
	ref = cache.DeduplicateResult("read_file", content, 3)
	if ref == "" {
		t.Error("expected reference string on duplicate, got empty")
	}
	if !strings.Contains(ref, "read_file") {
		t.Errorf("reference should mention tool name, got %q", ref)
	}
	if !strings.Contains(ref, "turn 1") {
		t.Errorf("reference should mention original turn, got %q", ref)
	}

	// 異なる内容は新規扱い
	ref = cache.DeduplicateResult("read_file", "different content", 4)
	if ref != "" {
		t.Errorf("expected empty string for different content, got %q", ref)
	}
}

func TestToolCache_DeduplicateResult_OnlyTargetTools(t *testing.T) {
	cache := NewToolCache()
	content := "some tool output"

	// 対象ツール: read_file, search_code, list_dir
	for _, tool := range []string{"read_file", "search_code", "list_dir"} {
		ref := cache.DeduplicateResult(tool, content+"_"+tool, 1)
		if ref != "" {
			t.Errorf("first call for %s should return empty, got %q", tool, ref)
		}
		ref = cache.DeduplicateResult(tool, content+"_"+tool, 2)
		if ref == "" {
			t.Errorf("second call for %s should return reference", tool)
		}
	}

	// 対象外ツール: bash, write_file, str_replace
	for _, tool := range []string{"bash", "write_file", "str_replace"} {
		ref := cache.DeduplicateResult(tool, "output_"+tool, 1)
		if ref != "" {
			t.Errorf("non-target tool %s should always return empty, got %q", tool, ref)
		}
		ref = cache.DeduplicateResult(tool, "output_"+tool, 2)
		if ref != "" {
			t.Errorf("non-target tool %s should always return empty on repeat, got %q", tool, ref)
		}
	}
}

func TestToolCache_DeduplicateResult_ClearResetsHashes(t *testing.T) {
	cache := NewToolCache()
	content := "file contents here"

	// 登録
	cache.DeduplicateResult("read_file", content, 1)

	// Clear で全リセット
	cache.Clear()

	// 同一内容でも新規扱い
	ref := cache.DeduplicateResult("read_file", content, 5)
	if ref != "" {
		t.Errorf("after Clear, same content should be treated as new, got %q", ref)
	}
}

func TestContentHash_Deterministic(t *testing.T) {
	content := "hello world"
	h1 := contentHash(content)
	h2 := contentHash(content)
	if h1 != h2 {
		t.Errorf("contentHash should be deterministic: %s != %s", h1, h2)
	}
	if len(h1) != 16 {
		t.Errorf("contentHash should be 16 hex chars, got %d: %s", len(h1), h1)
	}

	// 異なる内容は異なるハッシュ
	h3 := contentHash("different")
	if h1 == h3 {
		t.Error("different content should produce different hash")
	}
}

func TestIsDeduplicableToolResult(t *testing.T) {
	tests := []struct {
		tool string
		want bool
	}{
		{"read_file", true},
		{"search_code", true},
		{"list_dir", true},
		{"bash", false},
		{"write_file", false},
		{"str_replace", false},
		{"", false},
	}
	for _, tt := range tests {
		got := isDeduplicableToolResult(tt.tool)
		if got != tt.want {
			t.Errorf("isDeduplicableToolResult(%q) = %v, want %v", tt.tool, got, tt.want)
		}
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

func TestToolCache_ResultHashPruning(t *testing.T) {
	cache := NewToolCache()

	// MaxResultHashEntries + 1 個のエントリを登録
	for i := 0; i < MaxResultHashEntries+1; i++ {
		content := fmt.Sprintf("content-%d", i)
		cache.DeduplicateResult("read_file", content, i)
	}

	// resultHashes が MaxResultHashEntries 以下であることを確認
	cache.mu.RLock()
	count := len(cache.resultHashes)
	cache.mu.RUnlock()

	if count > MaxResultHashEntries {
		t.Fatalf("resultHashes should be capped at %d, got %d", MaxResultHashEntries, count)
	}

	// 最後に登録したエントリはまだ存在する（pruning は古いものから削除）
	lastContent := fmt.Sprintf("content-%d", MaxResultHashEntries)
	ref := cache.DeduplicateResult("read_file", lastContent, 999)
	if ref == "" {
		t.Error("most recent entry should still exist after pruning")
	}

	// 最初に登録したエントリは pruning で削除されている
	ref = cache.DeduplicateResult("read_file", "content-0", 999)
	if ref != "" {
		t.Errorf("oldest entry should have been pruned, got %q", ref)
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
