package search

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/searchcache"
)

func TestSinglePatternBundleCacheClearedWithSearchCache(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: "Close", Path: dir}
	ExecuteSearchCodeWithCache(cache, opts)

	if got := countSinglePatternBundleCacheEntries(); got == 0 {
		t.Fatal("expected bundle cache entry before clear")
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got == 0 {
		t.Fatal("expected affected-files cache entry before clear")
	}

	cache.ClearSearchCache()

	if got := countSinglePatternBundleCacheEntries(); got != 0 {
		t.Fatalf("expected bundle cache to be cleared, got %d entries", got)
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got != 0 {
		t.Fatalf("expected affected-files cache to be cleared, got %d entries", got)
	}
}

func TestSinglePatternBundleCacheInvalidatedWithFileInvalidation(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: "Close", Path: dir}
	normOpts, ok := normalizeSearchOptions(opts)
	if !ok {
		t.Fatal("expected normalized options")
	}
	normOpts.CtxLines = 3
	normOpts.TokenBudget = 15000
	cacheKey := buildSearchCacheKeyWithRoute(normOpts, planSearchRoute("Close", normOpts).cacheSignature())

	storeSinglePatternBundle("Close", cacheKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{Language: "go", Query: "Close", Canonical: "go|agent.go|5|Close", DisplayName: "Close", Kind: "function", File: "agent.go", Line: 5, EndLine: 7},
	})
	storeSinglePatternAffectedFiles("Close", cacheKey, []string{filepath.Join(dir, "agent.go")})
	otherKey := buildSearchCacheKeyWithRoute(normOpts, planSearchRoute("OtherClose", normOpts).cacheSignature())
	storeSinglePatternBundle("OtherClose", otherKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{Language: "go", Query: "OtherClose", Canonical: "go|other.go|5|OtherClose", DisplayName: "OtherClose", Kind: "function", File: "other.go", Line: 5, EndLine: 7},
	})
	storeSinglePatternAffectedFiles("OtherClose", otherKey, []string{filepath.Join(dir, "other.go")})
	cache.SetSearch("Close", cacheKey, "cached", []string{filepath.Join(dir, "agent.go")})
	cache.SetSearch("OtherClose", otherKey, "cached", []string{filepath.Join(dir, "other.go")})

	if got := countSinglePatternBundleCacheEntries(); got != 2 {
		t.Fatalf("expected 2 bundle cache entries before invalidate, got %d", got)
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got != 2 {
		t.Fatalf("expected 2 affected-files cache entries before invalidate, got %d", got)
	}

	cache.InvalidateSearchCacheForFile(filepath.Join(dir, "agent.go"))

	if got := countSinglePatternBundleCacheEntries(); got != 1 {
		t.Fatalf("expected targeted bundle cache invalidation, got %d entries", got)
	}
	if loadSinglePatternBundle("OtherClose", otherKey) == nil {
		t.Fatal("expected unrelated bundle cache entry to remain")
	}
	if loadSinglePatternAffectedFiles("Close", cacheKey) != nil {
		t.Fatal("expected targeted affected-files cache entry to be removed")
	}
	if loadSinglePatternAffectedFiles("OtherClose", otherKey) == nil {
		t.Fatal("expected unrelated affected-files cache entry to remain")
	}
}

func TestSinglePatternBundleCacheClearedOnSearchCacheEviction(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	storeSinglePatternBundle("keep", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "keep"}})
	storeSinglePatternBundle("drop", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "drop"}})

	if got := countSinglePatternBundleCacheEntries(); got != 2 {
		t.Fatalf("expected 2 bundle cache entries before eviction, got %d", got)
	}

	searchcache.NotifySearchCacheEvicted([]string{singlePatternBundleCacheKey("drop", "key")})

	if got := countSinglePatternBundleCacheEntries(); got != 1 {
		t.Fatalf("expected targeted bundle cache eviction, got %d entries", got)
	}
	if loadSinglePatternBundle("keep", "key") == nil {
		t.Fatal("expected unrelated bundle cache entry to remain after eviction")
	}
}

func TestSinglePatternBundleCachePreservesUnrelatedKeysOnTargetedInvalidation(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	storeSinglePatternBundle("keep", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "keep"}})
	storeSinglePatternBundle("drop", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "drop"}})

	searchcache.NotifySearchCacheInvalidatedKeys([]string{singlePatternBundleCacheKey("drop", "key")})

	if loadSinglePatternBundle("keep", "key") == nil {
		t.Fatal("expected unrelated bundle cache entry to remain after targeted invalidation")
	}
	if loadSinglePatternBundle("drop", "key") != nil {
		t.Fatal("expected targeted bundle cache entry to be removed")
	}
}
