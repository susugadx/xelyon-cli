package search

import (
	"path/filepath"
	"testing"
)

func TestTryStructuredImpactSearchResult_SingleStoresRouteBundleAndAffectedFiles(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	root := t.TempDir()
	writeStructuredImpactPipelineTestFile(t, filepath.Join(root, "run.go"), "package example\n\nfunc Run() {}\n")
	writeStructuredImpactPipelineTestFile(t, filepath.Join(root, "caller.go"), "package example\n\nfunc callRun() { Run() }\n")

	opts := SearchOptions{
		Pattern:            "Run",
		Path:               root,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}
	ctx := structuredImpactSearchContext{
		Pattern:  "Run",
		Route:    searchRouteTrace{Language: "go", InitialLane: searchLaneSymbol, SymbolQuery: "Run"},
		CacheKey: "structured-impact-cache-key",
	}
	cache := &testSearchCache{data: make(map[string]string)}
	bundle := newStructuredImpactPipelineTestBundle(root)

	scope := structuredImpactSameScope(opts)
	result, ok := tryStructuredImpactSearchResult(cache, ctx, scope, func(symbol string, scope structuredImpactScope) symbolResolveResult {
		if symbol != "Run" {
			t.Fatalf("resolver symbol = %q, want Run", symbol)
		}
		if scope.Definition.Path != root || scope.Evidence.Path != root {
			t.Fatalf("resolver scope = %+v, want definition/evidence path %q", scope, root)
		}
		return symbolResolveResult{
			Output: "resolver-output",
			Status: symbolResolveSingle,
			Bundle: bundle,
		}
	})
	if !ok {
		t.Fatal("expected structured impact result")
	}
	if result.Ambiguous {
		t.Fatal("Ambiguous = true, want false")
	}
	if result.Bundle == nil {
		t.Fatal("Bundle = nil, want runtime bundle")
	}
	if result.Rendered == "resolver-output" {
		t.Fatal("expected rendered output to be formatted from runtime bundle")
	}
	if !containsAffectedFile(result.AffectedFiles, filepath.Join(root, "run.go")) {
		t.Fatalf("expected affected files to include run.go, got %v", result.AffectedFiles)
	}

	cachedOutput, ok := cache.GetSearch(ctx.Pattern, ctx.CacheKey)
	if !ok {
		t.Fatal("expected search cache entry")
	}
	if cachedOutput != "resolver-output" {
		t.Fatalf("cached output = %q, want resolver output", cachedOutput)
	}

	storedBundle := loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey)
	if storedBundle == nil {
		t.Fatal("expected bundle cache entry")
	}
	if !storedBundle.Debug.Route.SymbolAttempted || !storedBundle.Debug.Route.SymbolResolved {
		t.Fatalf("stored route = %+v, want attempted and resolved", storedBundle.Debug.Route)
	}
	if storedBundle.Debug.Route.FinalLane != searchLaneSymbol {
		t.Fatalf("stored route final lane = %q, want %q", storedBundle.Debug.Route.FinalLane, searchLaneSymbol)
	}

	storedAffected := loadSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey)
	if !containsAffectedFile(storedAffected, filepath.Join(root, "run.go")) {
		t.Fatalf("expected stored affected files to include run.go, got %v", storedAffected)
	}
}
func TestTryStructuredImpactSearchResult_PassesDistinctDefinitionAndEvidenceScope(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	root := t.TempDir()
	writeStructuredImpactPipelineTestFile(t, filepath.Join(root, "run.go"), "package example\n\nfunc Run() {}\n")
	writeStructuredImpactPipelineTestFile(t, filepath.Join(root, "caller.go"), "package example\n\nfunc callRun() { Run() }\n")

	definition := SearchOptions{
		Pattern:            "Run",
		Path:               filepath.Join(root, "packages", "app", "src"),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}
	evidence := definition
	evidence.Path = root
	evidence.FilePattern = "packages/app/src/**/*.go"
	scope := structuredImpactScope{
		Definition: definition,
		Evidence:   evidence,
	}
	ctx := structuredImpactSearchContext{
		Pattern:  "Run",
		Route:    searchRouteTrace{Language: "go", InitialLane: searchLaneSymbol, SymbolQuery: "Run"},
		CacheKey: "structured-impact-scope-cache-key",
	}

	called := false
	result, ok := tryStructuredImpactSearchResult(&testSearchCache{data: make(map[string]string)}, ctx, scope, func(symbol string, got structuredImpactScope) symbolResolveResult {
		called = true
		if symbol != "Run" {
			t.Fatalf("resolver symbol = %q, want Run", symbol)
		}
		if got.Definition.Path != definition.Path {
			t.Fatalf("definition path = %q, want %q", got.Definition.Path, definition.Path)
		}
		if got.Evidence.Path != evidence.Path || got.Evidence.FilePattern != evidence.FilePattern {
			t.Fatalf("evidence scope = (%q, %q), want (%q, %q)", got.Evidence.Path, got.Evidence.FilePattern, evidence.Path, evidence.FilePattern)
		}
		return symbolResolveResult{
			Output: "resolver-output",
			Status: symbolResolveSingle,
			Bundle: newStructuredImpactPipelineTestBundle(root),
		}
	})
	if !ok || !called {
		t.Fatalf("structured impact result ok=%v called=%v", ok, called)
	}
	if result.Bundle == nil {
		t.Fatal("Bundle = nil, want runtime bundle")
	}
}
func TestTryStructuredImpactSearchResult_UsesRouteSymbolQuery(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	root := t.TempDir()
	writeStructuredImpactPipelineTestFile(t, filepath.Join(root, "run.go"), "package example\n\nfunc Run() {}\n")
	writeStructuredImpactPipelineTestFile(t, filepath.Join(root, "caller.go"), "package example\n\nfunc callRun() { Run() }\n")

	opts := SearchOptions{
		Pattern:       `Run\(`,
		Path:          root,
		InvocationCWD: root,
	}
	ctx := structuredImpactSearchContext{
		Pattern:  `Run\(`,
		Route:    searchRouteTrace{Language: "go", InitialLane: searchLaneSymbol, SymbolQuery: "Run", SymbolRescue: true},
		CacheKey: "structured-impact-rescue-cache-key",
	}

	called := false
	result, ok := tryStructuredImpactSearchResult(&testSearchCache{data: make(map[string]string)}, ctx, structuredImpactSameScope(opts), func(symbol string, scope structuredImpactScope) symbolResolveResult {
		called = true
		if symbol != "Run" {
			t.Fatalf("resolver symbol = %q, want routed SymbolQuery Run", symbol)
		}
		return symbolResolveResult{
			Output: "resolver-output",
			Status: symbolResolveSingle,
			Bundle: newStructuredImpactPipelineTestBundle(root),
		}
	})
	if !ok || !called {
		t.Fatalf("structured impact result ok=%v called=%v", ok, called)
	}
	if result.Bundle == nil {
		t.Fatal("Bundle = nil, want runtime bundle")
	}
}
