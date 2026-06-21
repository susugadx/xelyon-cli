package search

import (
	"path/filepath"
	"testing"
)

func TestTryStructuredImpactSearchResult_AmbiguousStoresAffectedFilesWithoutBundle(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	root := t.TempDir()
	buildPath := filepath.Join(root, "build.go")
	configPath := filepath.Join(root, "config.go")
	writeStructuredImpactPipelineTestFile(t, buildPath, "package example\n\nfunc Build() {}\n")
	writeStructuredImpactPipelineTestFile(t, configPath, "package example\n\ntype Config struct{}\nfunc (Config) Build() {}\n")

	opts := SearchOptions{
		Pattern:       "Build",
		Path:          root,
		InvocationCWD: root,
	}
	ctx := structuredImpactSearchContext{
		Pattern:  "Build",
		Route:    searchRouteTrace{Language: "go", InitialLane: searchLaneSymbol, SymbolQuery: "Build"},
		CacheKey: "structured-impact-ambiguous-cache-key",
	}
	cache := &testSearchCache{data: make(map[string]string)}
	output := `Multiple symbols matched "Build":`

	result, ok := tryStructuredImpactSearchResult(cache, ctx, structuredImpactSameScope(opts), func(symbol string, scope structuredImpactScope) symbolResolveResult {
		if symbol != "Build" {
			t.Fatalf("resolver symbol = %q, want Build", symbol)
		}
		return symbolResolveResult{
			Output:        output,
			Status:        symbolResolveMultiple,
			AffectedFiles: []string{buildPath, configPath},
		}
	})
	if !ok {
		t.Fatal("expected ambiguous structured impact result")
	}
	if !result.Ambiguous {
		t.Fatal("Ambiguous = false, want true")
	}
	if result.Bundle != nil {
		t.Fatalf("Bundle = %+v, want nil", result.Bundle)
	}
	if len(result.AffectedFiles) != 2 {
		t.Fatalf("AffectedFiles = %v, want two candidates", result.AffectedFiles)
	}

	cachedOutput, ok := cache.GetSearch(ctx.Pattern, ctx.CacheKey)
	if !ok {
		t.Fatal("expected ambiguous search cache entry")
	}
	if cachedOutput != output {
		t.Fatalf("cached output = %q, want ambiguous output", cachedOutput)
	}
	if loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey) != nil {
		t.Fatal("expected ambiguous result not to store bundle")
	}

	storedAffected := loadSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey)
	if !containsAffectedFile(storedAffected, buildPath) || !containsAffectedFile(storedAffected, configPath) {
		t.Fatalf("expected stored affected files to include ambiguous candidates, got %v", storedAffected)
	}
}
