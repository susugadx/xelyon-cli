package gathercontext

import "testing"

func TestParseRequestArgs_NormalizesInlineSearchScope(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": "A or B or review-harness in docs/",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "A or B or review-harness in docs/" {
		t.Fatalf("req.query = %q, want original query", req.query)
	}
	if req.searchQuery != "A,B,review-harness" {
		t.Fatalf("req.searchQuery = %q, want multi-pattern query", req.searchQuery)
	}
	if req.path != "" {
		t.Fatalf("req.path = %q, want no direct path scope", req.path)
	}
	if req.searchPath != "docs" {
		t.Fatalf("req.searchPath = %q, want inline docs scope", req.searchPath)
	}
	if !req.naturalSearchIntent {
		t.Fatal("expected natural search intent")
	}
}

func TestParseRequestArgs_StripsLeadingSearchIntentFromInlineScope(t *testing.T) {
	tests := []struct {
		query      string
		wantQuery  string
		wantPath   string
		wantSearch string
	}{
		{
			query:      "search for foo or quux in docs/",
			wantQuery:  "search for foo or quux in docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
		{
			query:      "search foo or quux in docs/",
			wantQuery:  "search foo or quux in docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
		{
			query:      "find foo or quux under docs/",
			wantQuery:  "find foo or quux under docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
		{
			query:      "look for foo or quux in docs/",
			wantQuery:  "look for foo or quux in docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
		{
			query:      "searching foo or quux under docs/",
			wantQuery:  "searching foo or quux under docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
		{
			query:      "searching for foo or quux under docs/",
			wantQuery:  "searching for foo or quux under docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
		{
			query:      "finding foo or quux in docs/",
			wantQuery:  "finding foo or quux in docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
		{
			query:      "looking for foo or quux in docs/",
			wantQuery:  "looking for foo or quux in docs/",
			wantPath:   "docs",
			wantSearch: "foo,quux",
		},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			req, errResult := parseRequestArgs(map[string]string{
				"query": tt.query,
			})
			if errResult != "" {
				t.Fatalf("unexpected parse error: %q", errResult)
			}
			if req.query != tt.wantQuery {
				t.Fatalf("req.query = %q, want original query", req.query)
			}
			if req.searchQuery != tt.wantSearch {
				t.Fatalf("req.searchQuery = %q, want normalized search query", req.searchQuery)
			}
			if req.searchPath != tt.wantPath {
				t.Fatalf("req.searchPath = %q, want inline scope", req.searchPath)
			}
			if !req.naturalSearchIntent {
				t.Fatal("expected natural search intent")
			}
		})
	}
}

func TestParseRequestArgs_DoesNotScopePlainPrepositionalSearch(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": "search for timeout in handler",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "search for timeout in handler" {
		t.Fatalf("req.query = %q, want original query", req.query)
	}
	if req.searchQuery != "timeout in handler" {
		t.Fatalf("req.searchQuery = %q, want unscoped search query", req.searchQuery)
	}
	if req.searchPath != "" {
		t.Fatalf("req.searchPath = %q, want no inline path scope", req.searchPath)
	}
	if !req.naturalSearchIntent {
		t.Fatal("expected natural search intent")
	}
}

func TestParseRequestArgs_NormalizesInlineScopeOnlyForPathLikeScope(t *testing.T) {
	tests := []struct {
		query    string
		wantPath string
	}{
		{query: "search for timeout in docs/", wantPath: "docs"},
		{query: "search for timeout in ./docs", wantPath: "./docs"},
		{query: "search for timeout in docs/handler", wantPath: "docs/handler"},
		{query: "search for timeout in handler.go", wantPath: "handler.go"},
		{query: "search for timeout in Makefile", wantPath: "Makefile"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			req, errResult := parseRequestArgs(map[string]string{
				"query": tt.query,
			})
			if errResult != "" {
				t.Fatalf("unexpected parse error: %q", errResult)
			}
			if req.searchQuery != "timeout" {
				t.Fatalf("req.searchQuery = %q, want stripped search query", req.searchQuery)
			}
			if req.searchPath != tt.wantPath {
				t.Fatalf("req.searchPath = %q, want path-like inline scope", req.searchPath)
			}
			if !req.naturalSearchIntent {
				t.Fatal("expected natural search intent")
			}
		})
	}
}

func TestParseRequestArgs_NormalizesAndInlineSearchScope(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": "A and B in docs/",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "A and B in docs/" {
		t.Fatalf("req.query = %q, want original query", req.query)
	}
	if req.searchQuery != "A and B" {
		t.Fatalf("req.searchQuery = %q, want scoped query", req.searchQuery)
	}
	if req.path != "" {
		t.Fatalf("req.path = %q, want no direct path scope", req.path)
	}
	if req.searchPath != "docs" {
		t.Fatalf("req.searchPath = %q, want inline docs scope", req.searchPath)
	}
	if !req.naturalSearchIntent {
		t.Fatal("expected natural search intent")
	}
}

func TestParseRequestArgs_ExplicitPathWinsOverInlineSearchScope(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": "A or B in docs/",
		"path":  "pkg",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "A or B in docs/" {
		t.Fatalf("req.query = %q, want original query", req.query)
	}
	if req.searchQuery != "A,B" {
		t.Fatalf("req.searchQuery = %q, want multi-pattern query", req.searchQuery)
	}
	if req.path != "pkg" {
		t.Fatalf("req.path = %q, want explicit path", req.path)
	}
	if req.searchPath != "pkg" {
		t.Fatalf("req.searchPath = %q, want explicit search path", req.searchPath)
	}
	if !req.naturalSearchIntent {
		t.Fatal("expected natural search intent")
	}
}
