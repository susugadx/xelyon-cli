package search

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func setupSearchTestMocks(t *testing.T) {
	t.Helper()
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
}

type testSearchCache struct {
	mu          sync.Mutex
	data        map[string]string
	affected    map[string][]string
	dataKeys    map[string]string
	getCalls    int
	setCalls    int
	lastGetPath string
	lastSetPath string
}

func (c *testSearchCache) GetFile(path string) (string, bool) { return "", false }
func (c *testSearchCache) SetFile(path, content string)       {}
func (c *testSearchCache) GetDir(path string) (string, bool)  { return "", false }
func (c *testSearchCache) SetDir(path, result string)         {}
func (c *testSearchCache) InvalidateFile(path string)         {}
func (c *testSearchCache) InvalidateDir(path string)          {}
func (c *testSearchCache) Clear()                             {}
func (c *testSearchCache) ClearSearchCache()                  { tools.NotifySearchCacheCleared() }
func (c *testSearchCache) InvalidateSearchCacheForFile(absPath string) {
	c.mu.Lock()
	deletedKeys := make([]string, 0)
	deleted := false
	for key, files := range c.affected {
		for _, fp := range files {
			if fp == absPath {
				if dataKey, ok := c.dataKeys[key]; ok {
					delete(c.data, dataKey)
					delete(c.dataKeys, key)
				}
				delete(c.affected, key)
				deleted = true
				deletedKeys = append(deletedKeys, key)
				break
			}
		}
	}
	c.mu.Unlock()
	if deleted {
		tools.NotifySearchCacheInvalidatedKeys(deletedKeys)
	}
}

func (c *testSearchCache) GetSearch(pattern, path string) (string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.getCalls++
	c.lastGetPath = path
	key := pattern + "|" + path
	if v, ok := c.data[key]; ok {
		return v, true
	}
	return "", false
}

func (c *testSearchCache) SetSearch(pattern, path, result string, affectedFiles []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setCalls++
	c.lastSetPath = path
	key := pattern + "|" + path
	c.data[key] = result
	if c.affected == nil {
		c.affected = make(map[string][]string)
	}
	if c.dataKeys == nil {
		c.dataKeys = make(map[string]string)
	}
	searchKey := singlePatternBundleCacheKey(pattern, path)
	c.affected[searchKey] = append([]string(nil), affectedFiles...)
	c.dataKeys[searchKey] = key
}

func TestSearchCode_CacheKeyUsesInternalTokenBudget(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "cached.go")
	if err := os.WriteFile(file1, []byte("func cached_target() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}

	result1 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, CtxLines: 0, TokenBudget: 500, IsRegex: true})
	if strings.Contains(result1, "No matches found") {
		t.Fatal("Expected matches on first search")
	}
	if !strings.Contains(cache.lastSetPath, "|3|15000|") {
		t.Fatalf("expected cache key to use internal defaults for context_lines=3 and token_budget=15000, got: %s", cache.lastSetPath)
	}

	result2 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, CtxLines: 0, TokenBudget: 99999, IsRegex: true})
	if result2 != result1 {
		t.Fatal("Expected second result to be served from the same cache key")
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected one cache write with normalized token budget, got %d", cache.setCalls)
	}
	if !strings.Contains(cache.lastGetPath, "|3|15000|") {
		t.Fatalf("expected cache lookup key to use internal defaults for context_lines=3 and token_budget=15000, got: %s", cache.lastGetPath)
	}
}

func TestSearchCode_AffectedFilesUseInvocationCWDForSubdirSearch(t *testing.T) {
	setupSearchTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(subdir, "target.go")
	if err := os.WriteFile(targetFile, []byte("package pkg\n\nconst target_text = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:            "target_text",
		Path:               ".",
		Mode:               string(SearchModeLiteral),
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "target.go") {
		t.Fatalf("expected text search result, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("target_text", cache.lastSetPath)
	affected := cache.affected[searchKey]
	if len(affected) != 1 || affected[0] != targetFile {
		t.Fatalf("expected affected files [%s], got %v", targetFile, affected)
	}

	cache.InvalidateSearchCacheForFile(targetFile)
	if _, ok := cache.GetSearch("target_text", cache.lastSetPath); ok {
		t.Fatal("expected text search cache entry to be invalidated after file edit")
	}
}
