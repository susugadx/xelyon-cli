package gathercontext

import (
	"io"
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
