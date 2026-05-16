package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTryStructuredImpactSearchResult_SingleStoresRouteBundleAndAffectedFiles(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

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
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

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
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

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

func TestTryStructuredImpactSearchResult_AmbiguousStoresAffectedFilesWithoutBundle(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

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

func TestTryStructuredImpactSearchResult_RejectsMalformedSingleBundle(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	root := t.TempDir()
	opts := SearchOptions{
		Pattern:       "Run",
		Path:          root,
		InvocationCWD: root,
	}
	ctx := structuredImpactSearchContext{
		Pattern:  "Run",
		Route:    searchRouteTrace{Language: "go", InitialLane: searchLaneSymbol, SymbolQuery: "Run"},
		CacheKey: "structured-impact-malformed-cache-key",
	}
	cache := &testSearchCache{data: make(map[string]string)}

	result, ok := tryStructuredImpactSearchResult(cache, ctx, structuredImpactSameScope(opts), func(symbol string, scope structuredImpactScope) symbolResolveResult {
		return symbolResolveResult{
			Output: "malformed-output",
			Status: symbolResolveSingle,
			Bundle: &SymbolBundle{
				Identity: SymbolBundleIdentity{
					Language:    "go",
					Query:       "Run",
					DisplayName: "Run",
					Kind:        "function",
					File:        "run.go",
					Line:        1,
				},
				Definition: SymbolBundleDefinition{File: "run.go", Line: 1},
			},
		}
	})
	if ok {
		t.Fatalf("ok = true, want malformed single bundle to fall back: %+v", result)
	}
}

func TestTryStructuredImpactSearchResult_CacheHitRejectsMalformedSingleBundle(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	root := t.TempDir()
	opts := SearchOptions{
		Pattern:       "Run",
		Path:          root,
		InvocationCWD: root,
	}
	ctx := structuredImpactSearchContext{
		Pattern:  "Run",
		Route:    searchRouteTrace{Language: "go", InitialLane: searchLaneSymbol, SymbolQuery: "Run"},
		CacheKey: "structured-impact-malformed-cache-hit-key",
	}
	cache := &testSearchCache{data: make(map[string]string)}
	cache.SetSearch(ctx.Pattern, ctx.CacheKey, "cached single output", nil)
	storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "go",
			Query:       "Run",
			DisplayName: "Run",
			Kind:        "function",
			File:        "run.go",
			Line:        1,
		},
		Definition: SymbolBundleDefinition{File: "run.go", Line: 1},
		Impact:     &SymbolBundleImpact{},
	})

	called := false
	result, ok := tryStructuredImpactSearchResult(cache, ctx, structuredImpactSameScope(opts), func(symbol string, scope structuredImpactScope) symbolResolveResult {
		called = true
		return symbolResolveResult{Status: symbolResolveNone}
	})
	if ok {
		t.Fatalf("ok = true, want malformed cached single bundle to fall back: %+v", result)
	}
	if !called {
		t.Fatal("resolver was not called after malformed cached single bundle")
	}
}

func newStructuredImpactPipelineTestBundle(root string) *SymbolBundle {
	return &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "go",
			Query:       "Run",
			Canonical:   "go|run.go|3|Run",
			DisplayName: "Run",
			Kind:        "function",
			File:        "run.go",
			Line:        3,
			EndLine:     3,
		},
		Definition: SymbolBundleDefinition{
			File:      "run.go",
			Line:      3,
			EndLine:   3,
			Signature: "func Run()",
			Body:      []string{"3: func Run() {}"},
		},
		Impact: &SymbolBundleImpact{
			RiskLevel: "low",
			RecommendedReads: []SymbolBundleItem{
				{Kind: "definition", File: "run.go", Line: 3, Snippet: "func Run()"},
				{Kind: "callers", File: "caller.go", Line: 3, Snippet: "Run()"},
			},
		},
		Debug: SymbolBundleDebug{
			FileRootPath: root,
		},
	}
}

func writeStructuredImpactPipelineTestFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
