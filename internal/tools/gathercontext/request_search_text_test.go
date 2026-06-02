package gathercontext

import "testing"

func TestParseRequestArgs_StripsLeadingSearchIntentFromUnscopedSearch(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": "search for foo or quux",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "search for foo or quux" {
		t.Fatalf("req.query = %q, want original query", req.query)
	}
	if req.searchQuery != "foo,quux" {
		t.Fatalf("req.searchQuery = %q, want normalized search query", req.searchQuery)
	}
	if !req.searchRouteIntent {
		t.Fatal("expected search route intent")
	}
}

func TestParseRequestArgs_PreservesQuotedOrPhrase(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantSearch string
		wantPath   string
	}{
		{
			name:       "unscoped quoted phrase",
			query:      `search for "foo or bar"`,
			wantSearch: "foo or bar",
		},
		{
			name:       "inline scoped quoted phrase",
			query:      `search for "foo or bar" in docs/`,
			wantSearch: "foo or bar",
			wantPath:   "docs",
		},
		{
			name:       "quoted phrase in or list",
			query:      `foo or "bar or baz"`,
			wantSearch: "foo,bar or baz",
		},
		{
			name:       "single quoted phrase in or list",
			query:      `foo or 'bar or baz'`,
			wantSearch: "foo,bar or baz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, errResult := parseRequestArgs(map[string]string{
				"query": tt.query,
			})
			if errResult != "" {
				t.Fatalf("unexpected parse error: %q", errResult)
			}
			if req.searchQuery != tt.wantSearch {
				t.Fatalf("req.searchQuery = %q, want %q", req.searchQuery, tt.wantSearch)
			}
			if req.searchPath != tt.wantPath {
				t.Fatalf("req.searchPath = %q, want %q", req.searchPath, tt.wantPath)
			}
			if !req.searchRouteIntent {
				t.Fatal("expected search route intent")
			}
		})
	}
}

func TestParseRequestArgs_PreservesPunctuationDelimitedOrLiterals(t *testing.T) {
	tests := []string{
		"error-or-warning",
		"foo/or/bar",
	}

	for _, query := range tests {
		t.Run(query, func(t *testing.T) {
			req, errResult := parseRequestArgs(map[string]string{
				"query": query,
			})
			if errResult != "" {
				t.Fatalf("unexpected parse error: %q", errResult)
			}
			if req.query != query {
				t.Fatalf("req.query = %q, want literal query unchanged", req.query)
			}
			if req.searchQuery != query {
				t.Fatalf("req.searchQuery = %q, want literal search query unchanged", req.searchQuery)
			}
			if req.searchRouteIntent {
				t.Fatal("punctuation-delimited literal should not be search route intent")
			}
		})
	}
}

func TestParseRequestArgs_PreservesCommaSeparatedSearchPatternsBeforeOrNormalization(t *testing.T) {
	query := "foo or bar,baz"
	req, errResult := parseRequestArgs(map[string]string{
		"query": query,
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != query {
		t.Fatalf("req.query = %q, want comma-separated query unchanged", req.query)
	}
	if req.searchQuery != query {
		t.Fatalf("req.searchQuery = %q, want comma-separated search patterns unchanged", req.searchQuery)
	}
	if !req.searchRouteIntent {
		t.Fatal("expected search route intent from first comma-separated entry")
	}
}
