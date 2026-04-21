package search

import (
	"strings"
	"testing"
)

func TestRenderMultiPatternOutput_DedupesGroupedSymbolBundleSection(t *testing.T) {
	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Canonical:   "example.Agent.Close",
			DisplayName: "Close",
			Kind:        "method",
			File:        "agent.go",
			Line:        3,
		},
		Definition: SymbolBundleDefinition{
			Line:      3,
			Signature: "func (a *Agent) Close() error",
		},
	}

	collected := []formattedPatternExecution{
		{
			Index: 0,
			singlePatternExecution: singlePatternExecution{
				Pattern: "Close",
				Output:  "symbol-close",
				Bundle:  bundle,
				Route: searchRouteTrace{
					SymbolCandidates: []string{"Close", "(*Agent).Close"},
				},
			},
		},
		{
			Index: 1,
			singlePatternExecution: singlePatternExecution{
				Pattern: "(*Agent).Close",
				Output:  "symbol-method-close",
				Bundle:  bundle,
				Route: searchRouteTrace{
					SymbolCandidates: []string{"(*Agent).Close"},
				},
			},
		},
		{
			Index: 2,
			singlePatternExecution: singlePatternExecution{
				Pattern: "run",
				Output:  "run.go:10 run()",
			},
		},
	}

	result := renderMultiPatternOutput(collected, []string{"Close", "(*Agent).Close", "run"}, SearchOptions{})
	if count := strings.Count(result, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected one grouped bundle section, got %d:\n%s", count, result)
	}
	if !strings.Contains(result, "Matched patterns: Close, (*Agent).Close") {
		t.Fatalf("expected matched patterns list in grouped bundle, got:\n%s", result)
	}
	if strings.Contains(result, "Pattern 1/3") || strings.Contains(result, "Pattern 2/3") {
		t.Fatalf("grouped bundle should replace individual pattern headers for bundled patterns, got:\n%s", result)
	}
	if !strings.Contains(result, `━━ Pattern 3/3: "run" ━━`) {
		t.Fatalf("expected non-bundled pattern section to remain, got:\n%s", result)
	}
	if !strings.HasSuffix(result, lineRangeHint) {
		t.Fatalf("expected lineRangeHint suffix, got:\n%s", result)
	}
}
