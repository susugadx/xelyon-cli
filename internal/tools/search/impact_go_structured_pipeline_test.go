package search

import "testing"

func TestNewStructuredGoImpactSearchContext_BuildsRouteScopedCacheKey(t *testing.T) {
	opts := SearchOptions{
		Pattern:  "Run",
		Intent:   "impact",
		Path:     ".",
		FileType: "go",
	}

	ctx, ok := newStructuredGoImpactSearchContext(opts)
	if !ok {
		t.Fatal("expected structured go impact context")
	}
	if ctx.Pattern != "Run" {
		t.Fatalf("pattern = %q, want %q", ctx.Pattern, "Run")
	}
	if ctx.Route.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want %q", ctx.Route.InitialLane, searchLaneSymbol)
	}

	wantKey := buildSearchCacheKeyWithRoute(opts, planSearchRoute("Run", opts).cacheSignature()+"|"+structuredGoImpactRouteTag)
	if ctx.CacheKey != wantKey {
		t.Fatalf("cache key = %q, want %q", ctx.CacheKey, wantKey)
	}
}

func TestNewStructuredGoImpactSearchContext_RejectsOutOfScopeInputs(t *testing.T) {
	tests := []struct {
		name string
		opts SearchOptions
	}{
		{
			name: "non go file type",
			opts: SearchOptions{
				Pattern:  "Run",
				Intent:   "impact",
				Path:     ".",
				FileType: "py",
			},
		},
		{
			name: "multi pattern",
			opts: SearchOptions{
				Pattern:  "Run,Build",
				Intent:   "impact",
				Path:     ".",
				FileType: "go",
			},
		},
		{
			name: "non impact intent",
			opts: SearchOptions{
				Pattern:  "Run",
				Intent:   "search",
				Path:     ".",
				FileType: "go",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := newStructuredGoImpactSearchContext(tt.opts); ok {
				t.Fatal("expected context creation to be rejected")
			}
		})
	}
}
