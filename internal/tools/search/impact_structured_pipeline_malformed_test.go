package search

import "testing"

func TestTryStructuredImpactSearchResult_RejectsMalformedSingleBundle(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

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
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

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
