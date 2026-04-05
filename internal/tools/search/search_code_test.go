package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCode_CacheKeyUsesInternalTokenBudget(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "cached.go")
	if err := os.WriteFile(file1, []byte("func cached_target() {}\n"), 0o644); err != nil {
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
