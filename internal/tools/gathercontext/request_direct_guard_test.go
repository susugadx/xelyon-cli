package gathercontext

import "testing"

func TestParseRequestArgs_PreservesDirectBatchContainingOrFilename(t *testing.T) {
	for _, query := range []string{
		"./red or blue.md,README.md",
		"README.md,./red or blue in docs.md",
		"./red or blue in docs.md,README.md",
	} {
		t.Run(query, func(t *testing.T) {
			req, errResult := parseRequestArgs(map[string]string{
				"query": query,
			})
			if errResult != "" {
				t.Fatalf("unexpected parse error: %q", errResult)
			}
			if req.query != query {
				t.Fatalf("req.query = %q, want direct batch unchanged", req.query)
			}
			if req.searchQuery != query {
				t.Fatalf("req.searchQuery = %q, want direct batch search fallback unchanged", req.searchQuery)
			}
			if req.naturalSearchIntent {
				t.Fatal("direct batch should not be natural search intent")
			}
		})
	}
}

func TestParseRequestArgs_PreservesExplicitPathContainingInlineScopeWords(t *testing.T) {
	for _, query := range []string{"./release notes in docs.md", "./A or B in docs/"} {
		t.Run(query, func(t *testing.T) {
			req, errResult := parseRequestArgs(map[string]string{
				"query": query,
			})
			if errResult != "" {
				t.Fatalf("unexpected parse error: %q", errResult)
			}
			if req.query != query {
				t.Fatalf("req.query = %q, want explicit path unchanged", req.query)
			}
			if req.searchQuery != query {
				t.Fatalf("req.searchQuery = %q, want explicit path search fallback unchanged", req.searchQuery)
			}
			if req.path != "" {
				t.Fatalf("req.path = %q, want empty path", req.path)
			}
			if req.naturalSearchIntent {
				t.Fatal("explicit path should not be natural search intent")
			}
		})
	}
}

func TestParseRequestArgs_PreservesDirectoryContainingWeakScopeWords(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": "files in docs/",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "files in docs/" {
		t.Fatalf("req.query = %q, want directory query unchanged", req.query)
	}
	if req.searchQuery != "files in docs/" {
		t.Fatalf("req.searchQuery = %q, want directory search fallback unchanged", req.searchQuery)
	}
	if req.path != "" {
		t.Fatalf("req.path = %q, want empty path", req.path)
	}
	if req.naturalSearchIntent {
		t.Fatal("directory query should not be natural search intent")
	}

	underReq, errResult := parseRequestArgs(map[string]string{
		"query": "files under docs/",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if underReq.query != "files under docs/" {
		t.Fatalf("under req.query = %q, want directory query unchanged", underReq.query)
	}
	if underReq.searchQuery != "files under docs/" {
		t.Fatalf("under req.searchQuery = %q, want directory search fallback unchanged", underReq.searchQuery)
	}
	if underReq.path != "" {
		t.Fatalf("under req.path = %q, want empty path", underReq.path)
	}
	if underReq.naturalSearchIntent {
		t.Fatal("under directory query should not be natural search intent")
	}

	orReq, errResult := parseRequestArgs(map[string]string{
		"query": "red or blue/",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if orReq.query != "red or blue/" {
		t.Fatalf("or req.query = %q, want directory query unchanged", orReq.query)
	}
	if orReq.searchQuery != "red or blue/" {
		t.Fatalf("or req.searchQuery = %q, want directory search fallback unchanged", orReq.searchQuery)
	}
	if orReq.path != "" {
		t.Fatalf("or req.path = %q, want empty path", orReq.path)
	}
	if orReq.naturalSearchIntent {
		t.Fatal("or directory query should not be natural search intent")
	}

	punctuationReq, errResult := parseRequestArgs(map[string]string{
		"query": "error-or-warning in docs/",
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if punctuationReq.query != "error-or-warning in docs/" {
		t.Fatalf("punctuation req.query = %q, want directory query unchanged", punctuationReq.query)
	}
	if punctuationReq.searchQuery != "error-or-warning in docs/" {
		t.Fatalf("punctuation req.searchQuery = %q, want directory search fallback unchanged", punctuationReq.searchQuery)
	}
	if punctuationReq.path != "" {
		t.Fatalf("punctuation req.path = %q, want empty path", punctuationReq.path)
	}
	if punctuationReq.naturalSearchIntent {
		t.Fatal("punctuation directory query should not be natural search intent")
	}
}
