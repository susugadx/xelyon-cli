package search

import "testing"

func TestNewStructuredGoImpactSearchContext_BuildsRouteScopedCacheKey(t *testing.T) {
	opts := SearchOptions{
		Pattern:  "Run",
		Intent:   "impact",
		Path:     ".",
		FileType: "go",
	}

	ctx, scope, ok := newStructuredGoImpactSearchContext(opts)
	if !ok {
		t.Fatal("expected structured go impact context")
	}
	if scope.Definition.FileType != "go" || scope.Definition.FilePattern != "" {
		t.Fatalf("definition opts file filter = (%q, %q), want (go, empty)", scope.Definition.FileType, scope.Definition.FilePattern)
	}
	if scope.Evidence.FileType != "go" || scope.Evidence.FilePattern != "" {
		t.Fatalf("evidence opts file filter = (%q, %q), want (go, empty)", scope.Evidence.FileType, scope.Evidence.FilePattern)
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

func TestNewStructuredGoImpactSearchContext_AllowsGoFilePatternRoute(t *testing.T) {
	dir := t.TempDir()
	opts := SearchOptions{
		Pattern:     "Run",
		Intent:      "impact",
		Path:        dir,
		FilePattern: "*.go",
	}

	ctx, scope, ok := newStructuredGoImpactSearchContext(opts)
	if !ok {
		t.Fatal("expected structured go impact context for *.go file pattern")
	}
	if ctx.Route.InitialLane != searchLaneSymbol {
		t.Fatalf("initial lane = %q, want %q", ctx.Route.InitialLane, searchLaneSymbol)
	}
	if ctx.Route.Language != "go" {
		t.Fatalf("route language = %q, want go", ctx.Route.Language)
	}
	if ctx.Route.SymbolRescue {
		t.Fatalf("SymbolRescue = true, want false for bare Go symbol with file pattern")
	}
	if ctx.Route.Decision != structuredGoImpactRouteTag {
		t.Fatalf("route decision = %q, want %q", ctx.Route.Decision, structuredGoImpactRouteTag)
	}
	if scope.Definition.FilePattern != "" || scope.Definition.FileType != "" {
		t.Fatalf("definition opts file filter = (%q, %q), want cleared file pattern", scope.Definition.FileType, scope.Definition.FilePattern)
	}
	if scope.Definition.Path != dir {
		t.Fatalf("definition path = %q, want original path %q", scope.Definition.Path, dir)
	}
	if scope.Evidence.FilePattern != "*.go" || scope.Evidence.FileType != "" {
		t.Fatalf("evidence opts file filter = (%q, %q), want (empty, *.go)", scope.Evidence.FileType, scope.Evidence.FilePattern)
	}
	if scope.Evidence.Path != dir {
		t.Fatalf("evidence path = %q, want original path %q", scope.Evidence.Path, dir)
	}
}

func TestNewStructuredGoImpactSearchContext_PreservesGoRescueRoute(t *testing.T) {
	opts := SearchOptions{
		Pattern:  `Target\(`,
		Intent:   "impact",
		Path:     ".",
		FileType: "go",
		Mode:     string(SearchModeAuto),
	}

	ctx, _, ok := newStructuredGoImpactSearchContext(opts)
	if !ok {
		t.Fatal("expected structured go impact context for rescued Go call pattern")
	}
	if ctx.Route.Decision != "go-rescue" {
		t.Fatalf("route decision = %q, want go-rescue", ctx.Route.Decision)
	}
	if !ctx.Route.SymbolRescue {
		t.Fatal("SymbolRescue = false, want true")
	}
	if ctx.Route.SymbolQuery != "Target" {
		t.Fatalf("symbol query = %q, want Target", ctx.Route.SymbolQuery)
	}
}

func TestNewStructuredGoImpactSearchContext_PreservesGoRescueRouteWithGoGlob(t *testing.T) {
	tests := []struct {
		name        string
		filePattern string
	}{
		{name: "basename glob", filePattern: "*.go"},
		{name: "scoped glob", filePattern: "src/**/*.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := SearchOptions{
				Pattern:     `Target\(`,
				Intent:      "impact",
				Path:        ".",
				FilePattern: tt.filePattern,
				Mode:        string(SearchModeAuto),
			}

			ctx, _, ok := newStructuredGoImpactSearchContext(opts)
			if !ok {
				t.Fatal("expected structured go impact context for rescued Go call pattern with glob filter")
			}
			if ctx.Route.Decision != "go-rescue" {
				t.Fatalf("route decision = %q, want go-rescue", ctx.Route.Decision)
			}
			if !ctx.Route.SymbolRescue {
				t.Fatal("SymbolRescue = false, want true")
			}
			if ctx.Route.SymbolQuery != "Target" {
				t.Fatalf("symbol query = %q, want Target", ctx.Route.SymbolQuery)
			}
		})
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
			if _, _, ok := newStructuredGoImpactSearchContext(tt.opts); ok {
				t.Fatal("expected context creation to be rejected")
			}
		})
	}
}
