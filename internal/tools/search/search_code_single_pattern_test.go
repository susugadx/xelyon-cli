package search

import (
	"path/filepath"
	"testing"
)

func TestLoadCachedSinglePatternExecution_Miss(t *testing.T) {
	ctx := newSinglePatternExecutionContext("Run", SearchOptions{Path: t.TempDir()})
	got, ok := loadCachedSinglePatternExecution(nil, ctx)
	if ok {
		t.Fatalf("expected cache miss, got hit: %+v", got)
	}
}

func TestLoadCachedSinglePatternExecution_HitUsesStoredAffectedFiles(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	dir := setupMultiLangDir(t, map[string]string{
		"run.go": "package example\n\nfunc Run() {}\n",
	})
	ctx := newSinglePatternExecutionContext("Run", SearchOptions{Path: dir, InvocationCWD: dir})
	cache := &testSearchCache{data: make(map[string]string)}

	wantPath := filepath.Join(dir, "run.go")
	cache.SetSearch(ctx.Pattern, ctx.CacheKey, "cached-output", nil)
	storeSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey, []string{wantPath})

	got, ok := loadCachedSinglePatternExecution(cache, ctx)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if got.Output != "cached-output" {
		t.Fatalf("expected cached output, got %q", got.Output)
	}
	if !containsAffectedFile(got.AffectedFiles, wantPath) {
		t.Fatalf("expected stored affected file %s, got %v", wantPath, got.AffectedFiles)
	}
}

func TestLoadCachedSinglePatternExecution_HitDerivesAffectedFilesFromBundle(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": "package example\n\nfunc Run() {}\n",
	})
	ctx := newSinglePatternExecutionContext("Run", SearchOptions{Path: dir, InvocationCWD: dir})
	cache := &testSearchCache{data: make(map[string]string)}

	cache.SetSearch(ctx.Pattern, ctx.CacheKey, "cached-output", nil)
	storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Canonical: "example.Run",
		},
		Definition: SymbolBundleDefinition{
			File: "pkg/run.go",
		},
		Debug: SymbolBundleDebug{
			FileRootPath: dir,
		},
	})

	got, ok := loadCachedSinglePatternExecution(cache, ctx)
	if !ok {
		t.Fatal("expected cache hit")
	}

	wantPath := filepath.Join(dir, "pkg", "run.go")
	if !containsAffectedFile(got.AffectedFiles, wantPath) {
		t.Fatalf("expected derived affected file %s, got %v", wantPath, got.AffectedFiles)
	}
}
