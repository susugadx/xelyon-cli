package gathercontext

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestPlanRoute_SearchAndLocatorContracts(t *testing.T) {
	root := setupRoutePlanFixtures(t)

	tests := []struct {
		name        string
		query       string
		path        string
		fileFilter  string
		wantKind    routeKind
		checkImpact bool
		wantImpact  bool
	}{
		{
			name:     "locator query uses compact read route",
			query:    "[L1,L2]",
			wantKind: routeLocatorRead,
		},
		{
			name:       "scoped soft basename honors stale file filter and stays searchable",
			query:      "README.md",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeSearch,
		},
		{
			name:       "scoped batch with bare file stays searchable",
			query:      "sample.go, nested.go:1-2",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeSearch,
		},
		{
			name:       "bare existing directory name stays searchable",
			query:      "config",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeSearch,
		},
		{
			name:     "existing slash directory without explicit marker stays searchable",
			query:    filepath.Join("internal", "tools"),
			wantKind: routeSearch,
		},
		{
			name:       "slash literal query stays searchable",
			query:      "pkg/errors",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeSearch,
		},
		{
			name:       "import-like slash literal stays searchable",
			query:      "github.com/foo/bar",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeSearch,
		},
		{
			name:        "single symbol query prefers structured impact",
			query:       "Builder",
			path:        "pkg",
			fileFilter:  "go",
			wantKind:    routeSearch,
			checkImpact: true,
			wantImpact:  true,
		},
		{
			name:        "qualified package symbol stays on search route",
			query:       "pkg.Func",
			path:        "pkg",
			fileFilter:  "go",
			wantKind:    routeSearch,
			checkImpact: true,
			wantImpact:  true,
		},
		{
			name:        "qualified method symbol stays on search route",
			query:       "Builder.Build",
			path:        "pkg",
			fileFilter:  "go",
			wantKind:    routeSearch,
			checkImpact: true,
			wantImpact:  true,
		},
		{
			name:       "module dotted symbol stays on search route",
			query:      "MyModule.Version",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeSearch,
		},
		{
			name:        "plain text query keeps auto search",
			query:       "close connection",
			path:        "pkg",
			fileFilter:  "go",
			wantKind:    routeSearch,
			checkImpact: true,
			wantImpact:  false,
		},
		{
			name:        "comma separated multi-pattern query does not force impact",
			query:       "Build,Run",
			path:        "pkg",
			fileFilter:  "go",
			wantKind:    routeSearch,
			checkImpact: true,
			wantImpact:  false,
		},
		{
			name:       "mixed direct and symbol query falls back to search",
			query:      "sample.go,pkg.Func",
			path:       "pkg",
			fileFilter: "go",
			wantKind:   routeSearch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildRoutePlan(newRoutePlanExecCtx(root), request{
				query:      tt.query,
				path:       tt.path,
				fileFilter: tt.fileFilter,
			})
			if plan.kind != tt.wantKind {
				t.Fatalf("plan.kind = %q, want %q", plan.kind, tt.wantKind)
			}
			switch plan.kind {
			case routeLocatorRead:
				if plan.locatorQuery != tt.query {
					t.Fatalf("plan.locatorQuery = %q, want %q", plan.locatorQuery, tt.query)
				}
			case routeSearch:
				if tt.checkImpact && plan.search.preferImpact != tt.wantImpact {
					t.Fatalf("plan.search.preferImpact = %v, want %v", plan.search.preferImpact, tt.wantImpact)
				}
				if plan.search.path != tt.path {
					t.Fatalf("plan.search.path = %q, want trimmed path %q", plan.search.path, tt.path)
				}
				if plan.search.fileFilter != tt.fileFilter {
					t.Fatalf("plan.search.fileFilter = %q, want trimmed filter %q", plan.search.fileFilter, tt.fileFilter)
				}
			}
		})
	}
}

func TestBuildRoutePlan_NaturalLanguageSearchScope(t *testing.T) {
	root := setupRoutePlanFixtures(t)

	tests := []struct {
		name      string
		query     string
		path      string
		wantQuery string
		wantPath  string
	}{
		{
			name:      "or list in docs becomes scoped multi pattern search",
			query:     "gather_context docs or review harness or review-harness or review command in docs/",
			wantQuery: "gather_context docs,review harness,review-harness,review command",
			wantPath:  "docs",
		},
		{
			name:      "under scope becomes scoped search",
			query:     "A or B under docs/",
			wantQuery: "A,B",
			wantPath:  "docs",
		},
		{
			name:      "and list in docs becomes scoped search",
			query:     "A and B in docs/",
			wantQuery: "A and B",
			wantPath:  "docs",
		},
		{
			name:      "search for prefix is not part of scoped search pattern",
			query:     "search for foo or quux in docs/",
			wantQuery: "foo,quux",
			wantPath:  "docs",
		},
		{
			name:      "find prefix is not part of scoped search pattern",
			query:     "find foo or quux under docs/",
			wantQuery: "foo,quux",
			wantPath:  "docs",
		},
		{
			name:      "look for prefix is not part of scoped search pattern",
			query:     "look for foo or quux in docs/",
			wantQuery: "foo,quux",
			wantPath:  "docs",
		},
		{
			name:      "searching prefix is not part of scoped search pattern",
			query:     "searching foo or quux under docs/",
			wantQuery: "foo,quux",
			wantPath:  "docs",
		},
		{
			name:      "finding prefix is not part of scoped search pattern",
			query:     "finding foo or quux in docs/",
			wantQuery: "foo,quux",
			wantPath:  "docs",
		},
		{
			name:      "looking for prefix is not part of scoped search pattern",
			query:     "looking for foo or quux in docs/",
			wantQuery: "foo,quux",
			wantPath:  "docs",
		},
		{
			name:      "plain prepositional search is not treated as path scope",
			query:     "search for timeout in handler",
			wantQuery: "timeout in handler",
			wantPath:  "",
		},
		{
			name:      "bare file name is treated as scoped search path",
			query:     "search for timeout in handler.go",
			wantQuery: "timeout",
			wantPath:  "handler.go",
		},
		{
			name:      "explicit path argument wins over inline scope",
			query:     "A or B in docs/",
			path:      "pkg",
			wantQuery: "A,B",
			wantPath:  "pkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := buildRoutePlan(newRoutePlanExecCtx(root), request{
				query: tt.query,
				path:  tt.path,
			})
			if plan.kind != routeSearch {
				t.Fatalf("plan.kind = %q, want %q", plan.kind, routeSearch)
			}
			if plan.search.query != tt.wantQuery {
				t.Fatalf("plan.search.query = %q, want %q", plan.search.query, tt.wantQuery)
			}
			if plan.search.path != tt.wantPath {
				t.Fatalf("plan.search.path = %q, want %q", plan.search.path, tt.wantPath)
			}
		})
	}
}

func TestBuildRoutePlan_PreservesQuotedOrPattern(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": `Search for the exact pattern "foo or bar" and report all file paths that contain it.`,
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}

	plan := buildRoutePlan(tools.ExecutionContext{}, req)
	if plan.kind != routeSearch {
		t.Fatalf("plan.kind = %q, want %q", plan.kind, routeSearch)
	}
	if plan.search.query != "foo or bar" {
		t.Fatalf("plan.search.query = %q, want exact quoted pattern", plan.search.query)
	}
}

func TestBuildRoutePlan_PreservesPunctuationDelimitedOrLiteral(t *testing.T) {
	for _, query := range []string{"error-or-warning", "foo/or/bar"} {
		t.Run(query, func(t *testing.T) {
			plan := buildRoutePlan(tools.ExecutionContext{}, request{query: query})
			if plan.kind != routeSearch {
				t.Fatalf("plan.kind = %q, want %q", plan.kind, routeSearch)
			}
			if plan.search.query != query {
				t.Fatalf("plan.search.query = %q, want literal query", plan.search.query)
			}
		})
	}
}

func TestBuildRoutePlan_PreservesCommaSeparatedSearchPatternsBeforeOrNormalization(t *testing.T) {
	query := "foo or bar,baz"
	plan := buildRoutePlan(tools.ExecutionContext{}, request{query: query})
	if plan.kind != routeSearch {
		t.Fatalf("plan.kind = %q, want %q", plan.kind, routeSearch)
	}
	if plan.search.query != query {
		t.Fatalf("plan.search.query = %q, want comma-separated search patterns unchanged", plan.search.query)
	}
}

func TestBuildRoutePlan_BracketedLocatorQueriesAlwaysUseLocatorRoute(t *testing.T) {
	tests := []string{
		"[L1]",
		"[l1]",
		"[l1, L2]",
	}

	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			plan := buildRoutePlan(tools.ExecutionContext{}, request{query: query})
			if plan.kind != routeLocatorRead {
				t.Fatalf("plan.kind = %q, want %q", plan.kind, routeLocatorRead)
			}
			if plan.locatorQuery != query {
				t.Fatalf("plan.locatorQuery = %q, want %q", plan.locatorQuery, query)
			}
		})
	}
}

func TestBuildRoutePlan_BareLocatorLikeQueriesNeedResolvedRegistryEntries(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		locations []locator.Location
		wantKind  routeKind
		wantQuery string
	}{
		{name: "bare single falls back without locator", query: "L1", wantKind: routeSearch},
		{name: "bare list falls back without locator", query: "L1,L2", wantKind: routeSearch},
		{name: "bare single uses locator route when resolved", query: "L1", locations: []locator.Location{{FilePath: "sample.txt", Line: 1, EndLine: 1}}, wantKind: routeLocatorRead, wantQuery: "L1"},
		{name: "bare list stays search when only one id resolves", query: "L1,L2", locations: []locator.Location{{FilePath: "sample.txt", Line: 1, EndLine: 1}}, wantKind: routeSearch},
		{name: "bare list uses locator route when all ids resolve", query: "L1,L2", locations: []locator.Location{{FilePath: "sample.txt", Line: 1, EndLine: 1}, {FilePath: "sample.txt", Line: 2, EndLine: 2}}, wantKind: routeLocatorRead, wantQuery: "L1,L2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCtx := tools.ExecutionContext{}
			if len(tt.locations) > 0 {
				reg := locator.NewRegistry()
				for _, loc := range tt.locations {
					reg.Register(loc)
				}
				execCtx.LocatorRegistry = reg
			}

			plan := buildRoutePlan(execCtx, request{query: tt.query})
			if plan.kind != tt.wantKind {
				t.Fatalf("plan.kind = %q, want %q", plan.kind, tt.wantKind)
			}
			if tt.wantKind == routeLocatorRead && plan.locatorQuery != tt.wantQuery {
				t.Fatalf("plan.locatorQuery = %q, want %q", plan.locatorQuery, tt.wantQuery)
			}
			if tt.wantKind == routeSearch && plan.search.query != tt.query {
				t.Fatalf("plan.search.query = %q, want %q", plan.search.query, tt.query)
			}
		})
	}
}

func TestBuildRoutePlan_TrimsScopedSearchInputs(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	plan := buildRoutePlan(newRoutePlanExecCtx(root), request{
		query:      "  Builder.Build  ",
		path:       "  pkg  ",
		fileFilter: "  go  ",
	})
	if plan.kind != routeSearch {
		t.Fatalf("plan.kind = %q, want %q", plan.kind, routeSearch)
	}
	if plan.search.query != "Builder.Build" {
		t.Fatalf("plan.search.query = %q, want %q", plan.search.query, "Builder.Build")
	}
	if plan.search.path != "pkg" {
		t.Fatalf("plan.search.path = %q, want %q", plan.search.path, "pkg")
	}
	if plan.search.fileFilter != "go" {
		t.Fatalf("plan.search.fileFilter = %q, want %q", plan.search.fileFilter, "go")
	}
}

func TestPlanRoute_QualifiedSymbolWithSearchScopeStaysOnSearchRoute(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	plan := buildRoutePlan(newRoutePlanExecCtx(root), request{
		query:      "Builder.Build",
		path:       "internal/agent",
		fileFilter: "go",
	})
	if plan.kind != routeSearch {
		t.Fatalf("plan.kind = %q, want %q", plan.kind, routeSearch)
	}
	if !plan.search.preferImpact {
		t.Fatal("qualified symbol query should still prefer impact search")
	}
}
