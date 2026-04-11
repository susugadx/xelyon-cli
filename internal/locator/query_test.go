package locator

import "testing"

func TestClassifyQueryPriority(t *testing.T) {
	reg := NewRegistry()
	reg.Register(Location{FilePath: "sample.txt", Line: 1, EndLine: 1})

	tests := []struct {
		name        string
		query       string
		registry    *Registry
		wantIntent  QueryIntent
		wantRoute   bool
		wantResolve bool
	}{
		{name: "strong bracketed single", query: "[L1]", registry: NewRegistry(), wantIntent: QueryIntentStrong, wantRoute: true},
		{name: "strong bracketed list", query: "[l1, L2]", registry: NewRegistry(), wantIntent: QueryIntentStrong, wantRoute: true},
		{name: "bare unresolved stays ambiguous", query: "L1", registry: NewRegistry(), wantIntent: QueryIntentAmbiguous, wantRoute: false},
		{name: "bare resolved single routes locator", query: "L1", registry: reg, wantIntent: QueryIntentAmbiguous, wantRoute: true, wantResolve: true},
		{name: "bare partially resolved list stays search", query: "L1,L2", registry: reg, wantIntent: QueryIntentAmbiguous, wantRoute: false, wantResolve: false},
		{name: "bare fully resolved list routes locator", query: "L1,L2", registry: func() *Registry {
			full := NewRegistry()
			full.Register(Location{FilePath: "sample.txt", Line: 1, EndLine: 1})
			full.Register(Location{FilePath: "sample.txt", Line: 2, EndLine: 2})
			return full
		}(), wantIntent: QueryIntentAmbiguous, wantRoute: true, wantResolve: true},
		{name: "non locator stays none", query: "Builder", registry: reg, wantIntent: QueryIntentNone, wantRoute: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ClassifyQueryPriority(tt.query, tt.registry)
			if got.Intent != tt.wantIntent {
				t.Fatalf("Intent = %v, want %v", got.Intent, tt.wantIntent)
			}
			if got.ShouldRouteLocator() != tt.wantRoute {
				t.Fatalf("ShouldRouteLocator() = %v, want %v", got.ShouldRouteLocator(), tt.wantRoute)
			}
			if got.HasResolvedLocator != tt.wantResolve {
				t.Fatalf("HasResolvedLocator = %v, want %v", got.HasResolvedLocator, tt.wantResolve)
			}
		})
	}
}
