package gathercontext

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestGatherContext_RequiresQuery(t *testing.T) {
	result, change, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}, map[string]string{"query": "   "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if change != nil {
		t.Fatalf("expected no file change, got %+v", change)
	}
	if !strings.Contains(result, "Error: query is required") {
		t.Fatalf("expected empty query error, got: %s", result)
	}
}

func TestGatherContext_RejectsCommaOnlyQuery(t *testing.T) {
	result, change, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout: io.Discard,
		Stderr: io.Discard,
	}, map[string]string{"query": "  ,  "})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if change != nil {
		t.Fatalf("expected no file change, got %+v", change)
	}
	if !strings.Contains(result, "Error: query must include at least one non-empty term") {
		t.Fatalf("expected comma-only query error, got: %s", result)
	}
}

func TestParseRequestArgs_AllowsEscapedLiteralCommaQuery(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{"query": `\,`})
	if errResult != "" {
		t.Fatalf("expected escaped literal comma query to stay valid, got %q", errResult)
	}
	if req.query != `\,` {
		t.Fatalf("req.query = %q, want escaped literal comma", req.query)
	}
}

func TestParseRequestArgs_NormalizesQuotedPatternInstruction(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": `Search for the exact pattern "func main" and report all file paths that contain it.`,
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "func main" {
		t.Fatalf("req.query = %q, want quoted pattern", req.query)
	}
}

func TestParseRequestArgs_NormalizesQuotedPatternField(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "colon",
			query: `pattern:"func main"`,
		},
		{
			name:  "space",
			query: `pattern "func main"`,
		},
		{
			name:  "colon with space",
			query: `pattern: "func main"`,
		},
		{
			name:  "equals",
			query: `pattern = "func main"`,
		},
		{
			name:  "single quote",
			query: `pattern:'func main'`,
		},
		{
			name:  "capitalized",
			query: `Pattern:"func main"`,
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
			if req.query != "func main" {
				t.Fatalf("req.query = %q, want quoted pattern", req.query)
			}
			if req.searchQuery != "func main" {
				t.Fatalf("req.searchQuery = %q, want quoted pattern", req.searchQuery)
			}
			if !req.literalSearchPattern {
				t.Fatal("expected literal search pattern metadata")
			}
			if !req.searchRouteIntent {
				t.Fatal("expected search route intent")
			}
		})
	}
}

func TestParseRequestArgs_KeepsAmbiguousQuotedPatternInstruction(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": `Search for patterns "func main" and "func init"`,
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != `Search for patterns "func main" and "func init"` {
		t.Fatalf("req.query = %q, want original ambiguous query", req.query)
	}
}

func TestParseRequestArgs_PreservesUnwrappedQuotedOrPatternAcrossNormalization(t *testing.T) {
	req, errResult := parseRequestArgs(map[string]string{
		"query": `Search for the exact pattern "foo or bar" and report all file paths that contain it.`,
	})
	if errResult != "" {
		t.Fatalf("unexpected parse error: %q", errResult)
	}
	if req.query != "foo or bar" {
		t.Fatalf("req.query = %q, want quoted pattern", req.query)
	}
	if !req.literalSearchPattern {
		t.Fatal("expected literal search pattern metadata")
	}
	if !req.searchRouteIntent {
		t.Fatal("expected search route intent")
	}
	if req.searchQuery != "foo or bar" {
		t.Fatalf("req.searchQuery = %q, want quoted pattern", req.searchQuery)
	}

	again := normalizeRequest(req)
	if again.query != "foo or bar" {
		t.Fatalf("renormalized query = %q, want exact quoted pattern", again.query)
	}
	if again.searchQuery != "foo or bar" {
		t.Fatalf("renormalized searchQuery = %q, want exact quoted pattern", again.searchQuery)
	}
	if !again.literalSearchPattern {
		t.Fatal("expected literal search pattern metadata to survive renormalization")
	}
	if !again.searchRouteIntent {
		t.Fatal("expected search route intent to survive renormalization")
	}
}

func TestGatherContext_SearchRouteNormalizesQuotedPatternInstruction(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "main.go"): "package main\n\nfunc main() {}\n",
	})

	for _, query := range []string{
		`Search for the exact pattern "func main" and report all file paths that contain it.`,
		`pattern:"func main"`,
		`pattern "func main"`,
	} {
		t.Run(query, func(t *testing.T) {
			result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
				"query":       query,
				"file_filter": "go",
			})

			assertGatherContextContainsAll(t, result, "Route: Auto search", "main.go")
		})
	}
}
