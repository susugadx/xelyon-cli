package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestBuildGoSymbolBundleIncludesEditSurface(t *testing.T) {
	bundle := buildGoSymbolBundle("Close", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:     "Close",
			Kind:     "method",
			File:     "agent.go",
			Line:     10,
			EndLine:  14,
			Receiver: "*Agent",
		},
		Body: []string{
			"10: func (a *Agent) Close() error {",
			"11: \treturn nil",
			"12: }",
		},
		Callers: []navigation.Reference{
			{File: "runner.go", Line: 22, Scope: "shutdown", Snippet: "return agent.Close()"},
		},
		TotalCallers: 1,
		Tests: []navigation.TestRef{
			{File: "agent_test.go", Line: 8, Name: "TestClose"},
		},
		TotalTests: 1,
	})
	result := formatSymbolBundle(bundle, nil, nil)

	if !strings.Contains(result, "Definition:") {
		t.Errorf("expected Definition section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers (1):") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
	if !strings.Contains(result, "Related Tests (1):") {
		t.Errorf("expected Related Tests section, got:\n%s", result)
	}
}

func TestBuildGoSymbolBundleCarriesDiagnostics(t *testing.T) {
	bundle := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Run",
			Kind:    "function",
			File:    "run.go",
			Line:    10,
			EndLine: 12,
		},
		Body: []string{
			"10: func Run() {",
			"11: }",
		},
		ResolvedViaLSP:     true,
		UpstreamIncomplete: true,
	})
	result := formatSymbolBundle(bundle, nil, nil)
	if !strings.Contains(result, "Warning: upstream search may be incomplete.") {
		t.Fatalf("expected incomplete warning in bundle output, got:\n%s", result)
	}
	if !strings.Contains(result, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in bundle output, got:\n%s", result)
	}
	if bundle.Diagnostics.Incomplete == nil || !*bundle.Diagnostics.Incomplete {
		t.Fatalf("Diagnostics.Incomplete = %v, want true", bundle.Diagnostics.Incomplete)
	}
	assertBoolPtr(t, bundle.Diagnostics.LSPAttempted, true, "Diagnostics.LSPAttempted")
	assertBoolPtr(t, bundle.Diagnostics.LSPAvailable, true, "Diagnostics.LSPAvailable")
	if bundle.Diagnostics.RawRefCount != nil || bundle.Diagnostics.AcceptedRefCount != nil || bundle.Diagnostics.DroppedRefCount != nil {
		t.Fatalf("reference counts = raw %v accepted %v dropped %v, want nil legacy counts", bundle.Diagnostics.RawRefCount, bundle.Diagnostics.AcceptedRefCount, bundle.Diagnostics.DroppedRefCount)
	}
	if bundle.Diagnostics.FallbackUsed != nil {
		t.Fatalf("Diagnostics.FallbackUsed = %v, want nil for legacy LSP-only diagnostics", *bundle.Diagnostics.FallbackUsed)
	}
}

func TestBuildGoSymbolBundleCarriesTruncatedDiagnostic(t *testing.T) {
	bundle := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Run",
			Kind:    "function",
			File:    "run.go",
			Line:    10,
			EndLine: 12,
		},
		Body: []string{
			"10: func Run() {",
			"11: }",
		},
		UpstreamTruncated: true,
	})
	result := formatSymbolBundle(bundle, nil, nil)
	if !strings.Contains(result, "Note: upstream results were truncated.") {
		t.Fatalf("expected truncation note in bundle output, got:\n%s", result)
	}
	if bundle.Diagnostics.Truncated == nil || !*bundle.Diagnostics.Truncated {
		t.Fatalf("Diagnostics.Truncated = %v, want true", bundle.Diagnostics.Truncated)
	}
}

func TestBuildGoSymbolBundleLeavesUnknownReferenceDiagnosticsUnset(t *testing.T) {
	bundle := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Run",
			Kind:    "function",
			File:    "run.go",
			Line:    10,
			EndLine: 12,
		},
		Body: []string{
			"10: func Run() {",
			"11: }",
		},
	})

	assertDiagnosticsResolvedBy(t, bundle.Diagnostics, symbolBundleResolvedByUnknown)
	if bundle.Diagnostics.LSPSource != "" {
		t.Fatalf("Diagnostics.LSPSource = %q, want empty for unknown reference diagnostics", bundle.Diagnostics.LSPSource)
	}
	if bundle.Diagnostics.LSPAttempted != nil || bundle.Diagnostics.LSPAvailable != nil || bundle.Diagnostics.FallbackUsed != nil {
		t.Fatalf("diagnostic bools = attempted %v available %v fallback %v, want nil unknowns", bundle.Diagnostics.LSPAttempted, bundle.Diagnostics.LSPAvailable, bundle.Diagnostics.FallbackUsed)
	}
	if bundle.Diagnostics.RawRefCount != nil || bundle.Diagnostics.AcceptedRefCount != nil || bundle.Diagnostics.DroppedRefCount != nil {
		t.Fatalf("reference counts = raw %v accepted %v dropped %v, want nil unknowns", bundle.Diagnostics.RawRefCount, bundle.Diagnostics.AcceptedRefCount, bundle.Diagnostics.DroppedRefCount)
	}
}

func TestFormatGoImpactSymbolBundleIncludesRiskRecommendedReadsAndOmittedCounts(t *testing.T) {
	bundle := buildGoSymbolBundleWithOptions("Builder", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:       "Builder",
			Kind:       "interface",
			File:       "builder.go",
			Line:       3,
			EndLine:    5,
			Exported:   true,
			PackageDir: ".",
		},
		Body: []string{
			"3: type Builder interface {",
			"4: \tBuild() string",
			"5: }",
		},
		Callers: []navigation.Reference{
			{File: "client.go", Line: 4, Scope: "UseBuilder", Snippet: "return b.Build()"},
			{File: "service.go", Line: 8, Scope: "UseServiceBuilder", Snippet: "return svc.Build()"},
		},
		TotalCallers: 4,
		Refs: []navigation.Reference{
			{File: "api/builder_test.go", Line: 5, Scope: "TestBuilder", Snippet: "_ = b.Build()", IsTest: true},
			{File: "cmd/demo/main.go", Line: 7, Scope: "main", Snippet: "_ = b.Build()"},
		},
		TotalRefs: 4,
		Tests: []navigation.TestRef{
			{File: "builder_test.go", Line: 4, Name: "TestBuilder"},
			{File: "api/builder_test.go", Line: 3, Name: "TestAPIBuilder"},
		},
		TotalTests: 3,
		Implementations: []navigation.ImplementationRef{
			{File: "builder_impl.go", Line: 3, Name: "FileBuilder"},
			{File: "builder_mock.go", Line: 3, Name: "MockBuilder"},
			{File: "builder_remote.go", Line: 3, Name: "RemoteBuilder"},
		},
	}, goSymbolBundleBuildOptions{
		implementationLimit: 2,
		impact: &SymbolBundleImpact{
			RiskLevel: "high",
			RecommendedReads: []SymbolBundleItem{
				{Kind: "definition", File: "builder.go", Line: 3, Snippet: "type Builder interface {", Name: "Builder"},
				{Kind: "callers", File: "client.go", Line: 4, Scope: "UseBuilder", Snippet: "return b.Build()"},
			},
		},
	})

	output := formatSymbolBundle(bundle, nil, nil)
	if !strings.Contains(output, "Risk: high") {
		t.Fatalf("expected risk line in impact bundle output, got:\n%s", output)
	}
	if !strings.Contains(output, "Recommended reads:") {
		t.Fatalf("expected recommended reads in impact bundle output, got:\n%s", output)
	}
	if !strings.Contains(output, "Omitted: callers +2, references +2, tests +1, implementations +1") {
		t.Fatalf("expected omitted counts summary in impact bundle output, got:\n%s", output)
	}
}

func TestBuildGoSymbolBundleCanonicalIsStableAcrossLineMoves(t *testing.T) {
	first := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run.go",
			Line:               10,
			EndLine:            12,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error"),
			StableKeyCollision: false,
		},
	})
	second := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run.go",
			Line:               40,
			EndLine:            42,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error"),
			StableKeyCollision: false,
		},
	})

	if first.Identity.Canonical == "" || second.Identity.Canonical == "" {
		t.Fatal("expected canonical identity to be populated")
	}
	if first.Identity.Canonical != second.Identity.Canonical {
		t.Fatalf("expected stable canonical identity across line moves, got %q vs %q", first.Identity.Canonical, second.Identity.Canonical)
	}
}

func TestBuildGoSymbolBundleCanonicalAddsFileDisambiguatorOnCollision(t *testing.T) {
	stableKey := stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error")
	first := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run_linux.go",
			Line:               10,
			EndLine:            12,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableKey,
			StableKeyCollision: true,
		},
	})
	second := buildGoSymbolBundle("Run", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:               "Run",
			Kind:               "method",
			File:               "pkg/run_darwin.go",
			Line:               10,
			EndLine:            12,
			Receiver:           "*Agent",
			ReceiverNorm:       "Agent",
			Signature:          "func (a *Agent) Run() error",
			PackageDir:         "pkg",
			StableKey:          stableKey,
			StableKeyCollision: true,
		},
	})

	if first.Identity.Canonical == second.Identity.Canonical {
		t.Fatalf("expected file disambiguator for colliding stable keys, got %q", first.Identity.Canonical)
	}
	if !strings.Contains(first.Identity.Canonical, "file=pkg/run_linux.go") {
		t.Fatalf("expected linux file disambiguator, got %q", first.Identity.Canonical)
	}
	if !strings.Contains(second.Identity.Canonical, "file=pkg/run_darwin.go") {
		t.Fatalf("expected darwin file disambiguator, got %q", second.Identity.Canonical)
	}
}
