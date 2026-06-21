package search

import "testing"

func TestStructuredImpactLanguageSpecs_DefineDispatchOrderAndPolicy(t *testing.T) {
	tests := []struct {
		name               string
		routeTag           string
		fileType           string
		routeLanguage      string
		expandSupplemental bool
	}{
		{
			name:          "go",
			routeTag:      structuredGoImpactRouteTag,
			fileType:      "go",
			routeLanguage: "go",
		},
		{
			name:               "typescript",
			routeTag:           structuredTypeScriptImpactRouteTag,
			fileType:           "ts",
			routeLanguage:      "js",
			expandSupplemental: true,
		},
		{
			name:               "javascript",
			routeTag:           structuredJavaScriptImpactRouteTag,
			fileType:           "js",
			routeLanguage:      "js",
			expandSupplemental: true,
		},
	}

	specs := structuredImpactLanguageSpecs()
	if len(specs) != len(tests) {
		t.Fatalf("structuredImpactLanguageSpecs() len = %d, want %d", len(specs), len(tests))
	}

	for i, tt := range tests {
		spec := specs[i]
		if spec.name != tt.name {
			t.Fatalf("spec[%d].name = %q, want %q", i, spec.name, tt.name)
		}
		if spec.routeTag != tt.routeTag {
			t.Fatalf("spec[%d].routeTag = %q, want %q", i, spec.routeTag, tt.routeTag)
		}
		if spec.expandSupplemental != tt.expandSupplemental {
			t.Fatalf("spec[%d].expandSupplemental = %v, want %v", i, spec.expandSupplemental, tt.expandSupplemental)
		}
		if spec.normalize == nil || spec.planRoute == nil || spec.resolver == nil {
			t.Fatalf("spec[%d] has nil function field: %+v", i, spec)
		}

		opts := SearchOptions{
			Pattern:  "Run",
			Intent:   "impact",
			Path:     ".",
			FileType: tt.fileType,
		}
		ctx, scope, ok := spec.newSearchContext(opts)
		if !ok {
			t.Fatalf("spec[%d].newSearchContext() rejected %s impact opts", i, tt.name)
		}
		if ctx.Route.Language != tt.routeLanguage {
			t.Fatalf("spec[%d] route language = %q, want %q", i, ctx.Route.Language, tt.routeLanguage)
		}
		wantKey := buildSearchCacheKeyWithRoute(opts, ctx.Route.cacheSignature()+"|"+tt.routeTag)
		if ctx.CacheKey != wantKey {
			t.Fatalf("spec[%d] cache key = %q, want %q", i, ctx.CacheKey, wantKey)
		}
		if scope.Definition.FileType != tt.fileType || scope.Evidence.FileType != tt.fileType {
			t.Fatalf("spec[%d] scope file type = (%q, %q), want %q", i, scope.Definition.FileType, scope.Evidence.FileType, tt.fileType)
		}
	}
}
