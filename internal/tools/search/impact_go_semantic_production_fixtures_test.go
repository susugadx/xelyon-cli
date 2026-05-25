package search

import (
	"fmt"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func buildStructuredGoProductionTestResult(t *testing.T, result navigation.InspectResult, plan impactplan.Plan) symbolResolveResult {
	t.Helper()
	symbol := "Build"
	if result.Symbol != nil && result.Symbol.Name != "" {
		symbol = result.Symbol.Name
	}
	resolved := buildStructuredGoImpactSingleSymbolResult(symbol, result, SearchOptions{}, SearchOptions{}, plan)
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("Status = %q, want single", resolved.Status)
	}
	if resolved.Bundle == nil {
		t.Fatal("Bundle = nil, want production bundle")
	}
	if resolved.Bundle.Impact == nil {
		t.Fatal("Impact = nil, want structured impact metadata")
	}
	return resolved
}

func goStructuredProductionFunctionFixture() navigation.InspectResult {
	result := semanticEvidenceGoBuildInspectFixture()
	result.Symbol.PackageDir = "pkg"
	result.TotalCallers = len(result.Callers)
	result.Refs = []navigation.Reference{{
		File:         "cmd/build/main.go",
		ResolvedPath: "/repo/cmd/build/main.go",
		Line:         12,
		Scope:        "main",
		Snippet:      "Build()",
	}}
	result.TotalRefs = len(result.Refs)
	result.TotalTests = len(result.Tests)
	return result
}

func goStructuredProductionImplementationFixture(total int) navigation.InspectResult {
	result := goStructuredProductionFunctionFixture()
	result.Symbol.Kind = "interface"
	result.Symbol.Signature = "type Builder interface { Build() string }"
	result.Implementations = make([]navigation.ImplementationRef, 0, total)
	for i := 1; i <= total; i++ {
		result.Implementations = append(result.Implementations, navigation.ImplementationRef{
			File:         fmt.Sprintf("pkg/impl_%02d.go", i),
			ResolvedPath: fmt.Sprintf("/repo/pkg/impl_%02d.go", i),
			Line:         20 + i,
			Name:         fmt.Sprintf("Implementation%d", i),
		})
	}
	return result
}

func goStructuredProductionDiagnosticFixture(diag navigation.InspectReferenceDiagnostics, truncated bool, incomplete bool, budgetHit bool) navigation.InspectResult {
	result := goStructuredProductionFunctionFixture()
	result.ReferenceDiagnostics = diag
	result.ResolvedViaLSP = diag.ResolvedBy == SymbolBundleResolvedByLSP
	result.UpstreamTruncated = truncated
	result.UpstreamIncomplete = incomplete
	if budgetHit {
		result.TotalCallers = len(result.Callers) + 2
		result.MoreCallers = true
	}
	return result
}
