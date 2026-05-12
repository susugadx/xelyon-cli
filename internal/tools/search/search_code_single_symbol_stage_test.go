package search

import (
	"path/filepath"
	"testing"
)

func TestResolveMultiSymbolAffectedFiles_DedupesResolvedAndOutputPaths(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go":  "package example\n\nfunc Run() {}\n",
		"pkg/test.go": "package example\n\nfunc TestRun() {}\n",
	})
	runPath := filepath.Join(dir, "pkg", "run.go")
	testPath := filepath.Join(dir, "pkg", "test.go")

	resolved := symbolResolveResult{
		Output:        "📄 pkg/test.go (1 match)",
		AffectedFiles: []string{runPath, runPath},
	}
	opts := SearchOptions{Path: dir, InvocationCWD: dir}

	got := resolveMultiSymbolAffectedFiles(resolved, opts)
	if len(got) != 2 {
		t.Fatalf("expected 2 deduped affected files, got %d (%v)", len(got), got)
	}
	if !containsAffectedFile(got, runPath) {
		t.Fatalf("expected resolved affected file %s, got %v", runPath, got)
	}
	if !containsAffectedFile(got, testPath) {
		t.Fatalf("expected output-derived affected file %s, got %v", testPath, got)
	}
}

func TestResolvedSinglePatternSymbolRoute(t *testing.T) {
	got := resolvedSinglePatternSymbolRoute(searchRouteTrace{
		InitialLane: searchLaneSymbol,
	})
	if got.FinalLane != searchLaneSymbol {
		t.Fatalf("expected final lane %q, got %q", searchLaneSymbol, got.FinalLane)
	}
	if !got.SymbolResolved {
		t.Fatal("expected symbol resolved route flag")
	}
}

func TestWriteSingleSymbolPatternCache_StoresBundleAndAffectedFiles(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	cache := &testSearchCache{data: make(map[string]string)}
	ctx := newSinglePatternExecutionContext("Run", SearchOptions{Path: t.TempDir()})
	resolved := symbolResolveResult{
		Output: "symbol-output",
		Bundle: &SymbolBundle{
			Identity: SymbolBundleIdentity{Canonical: "example.Run"},
		},
	}
	affected := []string{"/tmp/run.go"}

	writeSingleSymbolPatternCache(cache, ctx, resolved, affected)

	cachedOutput, ok := cache.GetSearch(ctx.Pattern, ctx.CacheKey)
	if !ok {
		t.Fatal("expected search cache entry for symbol result")
	}
	if cachedOutput != "symbol-output" {
		t.Fatalf("unexpected cached symbol output: %q", cachedOutput)
	}
	if loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey) == nil {
		t.Fatal("expected bundle cache entry to be stored")
	}
	storedAffected := loadSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey)
	if !containsAffectedFile(storedAffected, "/tmp/run.go") {
		t.Fatalf("expected stored affected files to include /tmp/run.go, got %v", storedAffected)
	}
}

func TestWriteSingleSymbolPatternCache_SkipsNilBundleStore(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	cache := &testSearchCache{data: make(map[string]string)}
	ctx := newSinglePatternExecutionContext("Run", SearchOptions{Path: t.TempDir()})

	writeSingleSymbolPatternCache(cache, ctx, symbolResolveResult{Output: "symbol-output"}, []string{"/tmp/run.go"})

	if loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey) != nil {
		t.Fatal("expected nil bundle to skip bundle cache storage")
	}
}
