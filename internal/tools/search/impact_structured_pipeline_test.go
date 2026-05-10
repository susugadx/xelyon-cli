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

	result, ok := tryStructuredImpactSearchResult(cache, ctx, opts, func(symbol string, opts SearchOptions) symbolResolveResult {
		if symbol != "Run" {
			t.Fatalf("resolver symbol = %q, want Run", symbol)
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

	result, ok := tryStructuredImpactSearchResult(cache, ctx, opts, func(symbol string, opts SearchOptions) symbolResolveResult {
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
