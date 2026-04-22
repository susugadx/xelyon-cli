package search

import (
	"fmt"
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

	result := renderMultiPatternOutput(collected, 3, SearchOptions{})
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

func TestAppendRenderedSection_AppendsMissingTrailingNewline(t *testing.T) {
	var sb strings.Builder
	appendRenderedSection(&sb, "first line")
	if got := sb.String(); got != "first line\n" {
		t.Fatalf("expected newline to be appended, got %q", got)
	}
}

func TestAppendRenderedSection_PreservesExistingTrailingNewline(t *testing.T) {
	var sb strings.Builder
	appendRenderedSection(&sb, "first line\n")
	appendRenderedSection(&sb, "second line")
	if got := sb.String(); got != "first line\nsecond line\n" {
		t.Fatalf("unexpected combined output: %s", fmt.Sprintf("%q", got))
	}
}

func TestPatternSymbolBundleCandidates_IncludesPatternAndRouteCandidates(t *testing.T) {
	execution := formattedPatternExecution{
		singlePatternExecution: singlePatternExecution{
			Pattern: "Close",
			Route: searchRouteTrace{
				SymbolCandidates: []string{"Close", "(*Agent).Close"},
			},
		},
	}

	got := patternSymbolBundleCandidates(execution)
	want := []string{"Close", "Close", "(*Agent).Close"}
	if len(got) != len(want) {
		t.Fatalf("unexpected candidate count: got=%d want=%d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected candidate[%d]: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestGroupPatternSymbolBundles_MergesPatternCandidatesByCanonical(t *testing.T) {
	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Canonical: "example.Agent.Close",
		},
	}
	collected := []formattedPatternExecution{
		{
			singlePatternExecution: singlePatternExecution{
				Pattern: "Close",
				Bundle:  bundle,
				Route: searchRouteTrace{
					SymbolCandidates: []string{"Close", "(*Agent).Close"},
				},
			},
		},
		{
			singlePatternExecution: singlePatternExecution{
				Pattern: "(*Agent).Close",
				Bundle:  bundle,
				Route: searchRouteTrace{
					SymbolCandidates: []string{"(*Agent).Close", "Close"},
				},
			},
		},
		{
			singlePatternExecution: singlePatternExecution{
				Pattern: "run",
			},
		},
	}

	groups := groupPatternSymbolBundles(collected)
	group, ok := groups["example.Agent.Close"]
	if !ok {
		t.Fatalf("expected canonical group to exist: %#v", groups)
	}

	want := []string{"Close", "(*Agent).Close"}
	if len(group.Patterns) != len(want) {
		t.Fatalf("unexpected grouped patterns: got=%v want=%v", group.Patterns, want)
	}
	for i := range want {
		if group.Patterns[i] != want[i] {
			t.Fatalf("unexpected grouped pattern[%d]: got=%q want=%q", i, group.Patterns[i], want[i])
		}
	}
}
