package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchCode_CacheHitAfterSearch(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "cached.go")
	if err := os.WriteFile(file1, []byte("func cached_target() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}

	result1 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if strings.Contains(result1, "No matches found") {
		t.Fatal("Expected matches on first search")
	}

	if cache.setCalls == 0 {
		t.Error("Expected SetSearch to be called after first search")
	}

	getCalls := cache.getCalls
	result2 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, FilePattern: "", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})

	if cache.getCalls <= getCalls {
		t.Error("Expected GetSearch to be called on second search")
	}
	if result2 != result1 {
		t.Error("Expected same result from cache")
	}
}
