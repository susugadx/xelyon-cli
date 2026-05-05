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

func TestGatherContext_SearchRouteNormalizesQuotedPatternInstruction(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "main.go"): "package main\n\nfunc main() {}\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       `Search for the exact pattern "func main" and report all file paths that contain it.`,
		"file_filter": "go",
	})

	assertGatherContextContainsAll(t, result, "Route: Auto search", "main.go")
}
