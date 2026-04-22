package navigation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func TestFormatMultipleCandidates(t *testing.T) {
	candidates := []SymbolCandidate{
		{Name: "Build", Kind: "function", File: "internal/agent/agent.go", Line: 21, EndLine: 85},
		{Name: "Build", Kind: "method", File: "internal/config/config.go", Line: 88, EndLine: 120, Receiver: "*Config"},
	}
	result := formatMultipleCandidates("Build", candidates, nil)
	if !strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected multiple candidates header, got: %s", result)
	}
	if !strings.Contains(result, "agent.go") {
		t.Errorf("expected agent.go in output, got: %s", result)
	}
	if !strings.Contains(result, "Refine with path") {
		t.Errorf("expected path hint, got: %s", result)
	}
}

func TestFormatMultipleCandidates_UsesQualifiedMethodNames(t *testing.T) {
	candidates := []SymbolCandidate{
		{Name: "Build", Kind: "function", File: "internal/agent/agent.go", Line: 21, EndLine: 85},
		{Name: "Build", Kind: "method", File: "internal/config/config.go", Line: 88, EndLine: 120, Receiver: "*Config"},
	}

	result := formatMultipleCandidates("Build", candidates, nil)
	if !strings.Contains(result, "(*Config).Build") {
		t.Fatalf("expected receiver-qualified method name in candidate list, got: %s", result)
	}
}

func TestFormatInspectResult(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body: []string{
			"21: func (a *Agent) Build() error {",
			"22:     return nil",
			"23: }",
		},
		Callers: []Reference{
			{File: "cmd/root.go", Line: 88, Scope: "func main"},
		},
		Refs: []Reference{
			{File: "init.go", Line: 30, Snippet: "var defaultBuilder = Build"},
		},
		Tests: []TestRef{
			{File: "agent_test.go", Name: "TestBuild_Normal", Line: 55},
		},
	}, nil)

	if !strings.Contains(result, "── function Build") {
		t.Error("expected header")
	}
	if !strings.Contains(result, "Callers (1)") {
		t.Error("expected callers section")
	}
	if !strings.Contains(result, "References (1)") {
		t.Error("expected references section")
	}
	if !strings.Contains(result, "Related tests (1)") {
		t.Error("expected tests section")
	}
}

func TestFormatInspectResult_LocatorsPreferResolvedPathsForRelatedItems(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootShadow := filepath.Join(root, "target.go")
	subdirTarget := filepath.Join(subdir, "target.go")
	subdirTest := filepath.Join(subdir, "target_test.go")
	for _, path := range []string{rootShadow, subdirTarget, subdirTest} {
		if err := os.WriteFile(path, []byte("package pkg\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	reg := locator.NewRegistry()
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name:     "Build",
			Kind:     "function",
			File:     "pkg/source.go",
			Line:     10,
			EndLine:  20,
			RootPath: root,
		},
		Body: []string{"10: func Build() {}"},
		Callers: []Reference{
			{File: "target.go", ResolvedPath: subdirTarget, Line: 3, Scope: "func main"},
		},
		Refs: []Reference{
			{File: "target.go", ResolvedPath: subdirTarget, Line: 4, Snippet: "Build()"},
		},
		Tests: []TestRef{
			{File: "target_test.go", ResolvedPath: subdirTest, Name: "TestBuild", Line: 5},
		},
		Implementations: []ImplementationRef{
			{File: "target.go", ResolvedPath: subdirTarget, Line: 6, Name: "Builder"},
		},
	}, reg)

	for _, needle := range []string{"target.go:3", "target.go:4 | Build()", "target_test.go:5 | func TestBuild", "target.go:6 Builder"} {
		if !strings.Contains(result, needle) {
			t.Fatalf("expected %q in output, got:\n%s", needle, result)
		}
	}

	for _, id := range []string{"[L2]", "[L3]", "[L4]", "[L5]"} {
		loc, ok := reg.Resolve(id)
		if !ok {
			t.Fatalf("expected locator %s to resolve", id)
		}
		if loc.ResolvedPath == rootShadow {
			t.Fatalf("expected locator %s to avoid root shadow path, got %+v", id, loc)
		}
	}
	if loc, _ := reg.Resolve("[L2]"); loc.ResolvedPath != subdirTarget {
		t.Fatalf("expected caller locator to use %s, got %+v", subdirTarget, loc)
	}
	if loc, _ := reg.Resolve("[L4]"); loc.ResolvedPath != subdirTest {
		t.Fatalf("expected test locator to use %s, got %+v", subdirTest, loc)
	}
}

func TestFormatInspectResult_SummaryContractForTruncatedSections(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body: []string{"21: func Build() {}"},
		Callers: []Reference{
			{File: "cmd/root.go", Line: 88, Scope: "func main"},
			{File: "cmd/serve.go", Line: 42, Scope: "func runServer"},
		},
		TotalCallers: 5,
		MoreCallers:  true,
		Refs: []Reference{
			{File: "init.go", Line: 30, Snippet: "var defaultBuilder = Build"},
		},
		TotalRefs: 3,
		MoreRefs:  true,
		Tests: []TestRef{
			{File: "agent_test.go", Name: "TestBuild_Normal", Line: 55},
		},
		TotalTests: 2,
		MoreTests:  true,
	}, nil)

	if !strings.Contains(result, "Callers: 2 examples (of 5 found)") {
		t.Errorf("expected callers shown/total contract, got: %s", result)
	}
	if !strings.Contains(result, "References: 1 examples (of 3 found)") {
		t.Errorf("expected refs shown/total contract, got: %s", result)
	}
	if !strings.Contains(result, "Related tests: 1 examples (of 2 found)") {
		t.Errorf("expected tests shown/total contract, got: %s", result)
	}
	if !strings.Contains(result, "(+ more callers. Use search_code for more results)") {
		t.Errorf("expected callers next action hint, got: %s", result)
	}
	if !strings.Contains(result, "(+ more references. Use search_code for more results)") {
		t.Errorf("expected refs next action hint, got: %s", result)
	}
	if !strings.Contains(result, "(+ more tests. Use search_code for more results)") {
		t.Errorf("expected tests next action hint, got: %s", result)
	}
	if strings.Contains(result, "has related tests") {
		t.Errorf("legacy placeholder style must not appear, got: %s", result)
	}
	if !strings.Contains(result, "agent_test.go:55 | func TestBuild_Normal") {
		t.Errorf("expected related test path/line/name, got: %s", result)
	}
}

func TestFormatInspectResult_NextActionHints_OnlyWhenMore(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body: []string{"21: func Build() {}"},
		Callers: []Reference{
			{File: "cmd/root.go", Line: 88, Scope: "func main"},
		},
		Refs: []Reference{
			{File: "init.go", Line: 30, Snippet: "var defaultBuilder = Build"},
		},
		Tests: []TestRef{
			{File: "agent_test.go", Name: "TestBuild_Normal", Line: 55},
		},
	}, nil)

	if strings.Contains(result, "examples (of") {
		t.Errorf("shown/total format must not appear when section is complete, got: %s", result)
	}
	if strings.Contains(result, "(+ more callers") ||
		strings.Contains(result, "(+ more references") ||
		strings.Contains(result, "(+ more tests") {
		t.Errorf("next action hints must appear only when More* is true, got: %s", result)
	}
}

func TestFormatInspectResult_UpstreamIncompletePrecedenceOverTruncated(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:               []string{"21: func Build() {}"},
		UpstreamIncomplete: true,
		UpstreamTruncated:  true,
	}, nil)

	if !strings.Contains(result, "incomplete due to errors") {
		t.Errorf("expected incomplete warning, got: %s", result)
	}
	if strings.Contains(result, "truncated upstream") {
		t.Errorf("truncated note must not be emitted when incomplete warning is shown, got: %s", result)
	}
}
func TestFormatInspectResult_UpstreamTruncated(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:              []string{"21: func Build() {}"},
		UpstreamTruncated: true,
	}, nil)

	if !strings.Contains(result, "truncated upstream") {
		t.Errorf("expected upstream truncation notice, got: %s", result)
	}
}

// Regression 5: 上流検索が打ち切られていない場合は表示されない
func TestFormatInspectResult_UpstreamNotTruncated(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:              []string{"21: func Build() {}"},
		UpstreamTruncated: false,
	}, nil)

	if strings.Contains(result, "truncated upstream") {
		t.Errorf("expected no upstream line when not truncated, got: %s", result)
	}
}

// Regression 6: 上流検索の異常終了が警告として出力される
func TestFormatInspectResult_UpstreamIncomplete(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:               []string{"21: func Build() {}"},
		UpstreamIncomplete: true,
	}, nil)

	if !strings.Contains(result, "incomplete due to errors") {
		t.Errorf("expected upstream incomplete notice, got: %s", result)
	}
}

// Regression 7: 上流検索が正常完了した場合は警告が出ない
func TestFormatInspectResult_UpstreamComplete(t *testing.T) {
	result := formatInspectResult(InspectResult{
		Symbol: &SymbolCandidate{
			Name: "Build", Kind: "function",
			File: "agent.go", Line: 21, EndLine: 85,
		},
		Body:               []string{"21: func Build() {}"},
		UpstreamIncomplete: false,
	}, nil)

	if strings.Contains(result, "truncated upstream") {
		t.Errorf("expected no incomplete warning when search is complete, got: %s", result)
	}
}
