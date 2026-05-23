package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestExecuteSearchCode_ImpactIntentStructuredGoWidensRiskOnTruncationSignals(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"helper.go": `package example

func helper() {}

func a() { helper() }
func b() { helper() }
func c() { helper() }
func d() { helper() }
func e() { helper() }
`,
	})

	lspClient := &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "helper.go", Line: 5, Character: 12, EndLine: 5, EndChar: 18}, {File: "helper.go", Line: 6, Character: 12, EndLine: 6, EndChar: 18}, {File: "helper.go", Line: 7, Character: 12, EndLine: 7, EndChar: 18}, {File: "helper.go", Line: 8, Character: 12, EndLine: 8, EndChar: 18}, {File: "helper.go", Line: 9, Character: 12, EndLine: 9, EndChar: 18}}}
	output := ExecuteSearchCode(SearchOptions{
		Pattern:   "helper",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: lspClient,
	})

	if strings.Contains(output, "Pattern 1/") {
		t.Fatalf("expected structured impact path, got:\n%s", output)
	}
	if !strings.Contains(output, "Risk: medium") {
		t.Fatalf("expected truncation signal to widen structured impact risk to at least medium, got:\n%s", output)
	}
	if lspClient.findReferencesCalls != 1 {
		t.Fatalf("FindReferences calls = %d, want 1 collect pass", lspClient.findReferencesCalls)
	}
}

func TestClassifyGoImpactRisk_WidensOnUpstreamSignals(t *testing.T) {
	base := navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:       "helper",
			Kind:       "function",
			File:       "helper.go",
			Line:       3,
			EndLine:    3,
			PackageDir: ".",
		},
	}

	truncated := base
	truncated.UpstreamTruncated = true
	if got := classifyGoImpactRisk(truncated); got != impactplan.RiskMedium {
		t.Fatalf("classifyGoImpactRisk(truncated) = %q, want %q", got, impactplan.RiskMedium)
	}

	incomplete := base
	incomplete.UpstreamIncomplete = true
	if got := classifyGoImpactRisk(incomplete); got != impactplan.RiskMedium {
		t.Fatalf("classifyGoImpactRisk(incomplete) = %q, want %q", got, impactplan.RiskMedium)
	}
}
