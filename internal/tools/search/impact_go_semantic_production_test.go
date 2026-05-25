package search

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestStructuredGoImpactSemanticProductionBuildsFunctionBundle(t *testing.T) {
	plan := impactplan.PlanForRisk(impactplan.RiskHigh)
	result := goStructuredProductionFunctionFixture()
	resolved := buildStructuredGoProductionTestResult(t, result, plan)
	bundle := resolved.Bundle

	if bundle.Identity.Language != "go" || bundle.Identity.DisplayName != "Build" || bundle.Identity.Canonical != canonicalGoSymbolBundleKey(*result.Symbol) {
		t.Fatalf("identity = %+v, want stable Go Build identity", bundle.Identity)
	}
	if bundle.Definition.File != result.Symbol.File || bundle.Definition.Line != result.Symbol.Line || bundle.Definition.Signature != result.Body[0] {
		t.Fatalf("definition = %+v, want Go definition with body[0] signature %q", bundle.Definition, result.Body[0])
	}
	if !reflect.DeepEqual(bundle.Definition.Body, result.Body) {
		t.Fatalf("definition body = %+v, want %+v", bundle.Definition.Body, result.Body)
	}
	if !reflect.DeepEqual(bundle.Impact, buildGoImpactMetadata(result, plan.RiskLevel)) {
		t.Fatalf("impact = %+v, want buildGoImpactMetadata result", bundle.Impact)
	}
	assertDiagnosticsEqualToGoSourceOfTruth(t, result, bundle)
	assertGoStructuredProductionSectionOrder(t, bundle,
		SemanticReferenceSectionKindCallers,
		SemanticReferenceSectionKindReferences,
		SemanticReferenceSectionKindTests,
		SemanticReferenceSectionKindImplementations,
	)
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 0, "definition", "pkg/build.go")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 1, "callers", "pkg/app.go")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 2, "tests", "pkg/build_test.go")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 3, "implementations", "pkg/impl.go")
	assertRecommendedReadAt(t, bundle.Impact.RecommendedReads, 4, "references", "cmd/build/main.go")
}

func TestStructuredGoImpactSemanticProductionDefinitionSignatureMatchesLegacyGoBundle(t *testing.T) {
	plan := impactplan.PlanForRisk(impactplan.RiskLow)
	result := goStructuredProductionFunctionFixture()

	withBody := buildStructuredGoProductionTestResult(t, result, plan)
	if got := withBody.Bundle.Definition.Signature; got != result.Body[0] {
		t.Fatalf("definition signature with body = %q, want body[0] %q", got, result.Body[0])
	}

	result.Body = nil
	withoutBody := buildStructuredGoProductionTestResult(t, result, plan)
	if got := withoutBody.Bundle.Definition.Signature; got != "" {
		t.Fatalf("definition signature without body = %q, want empty", got)
	}
	if got := withoutBody.Bundle.Impact.RecommendedReads[0].Snippet; got != result.Symbol.Signature {
		t.Fatalf("definition recommended read snippet = %q, want symbol signature %q", got, result.Symbol.Signature)
	}
}

func TestStructuredGoImpactSemanticProductionPreservesMethodStableIdentity(t *testing.T) {
	stableKey := stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error")
	plan := impactplan.PlanForRisk(impactplan.RiskMedium)
	first := buildStructuredGoProductionTestResult(t, semanticEvidenceGoRunInspectFixture(stableKey), plan)

	moved := semanticEvidenceGoRunInspectFixture(stableKey)
	movedSymbol := *moved.Symbol
	movedSymbol.Line = 40
	movedSymbol.EndLine = 42
	moved.Symbol = &movedSymbol
	second := buildStructuredGoProductionTestResult(t, moved, plan)

	if first.Bundle.Identity.DisplayName != "(*Agent).Run" {
		t.Fatalf("display name = %q, want (*Agent).Run", first.Bundle.Identity.DisplayName)
	}
	if !strings.Contains(first.Output, "method (*Agent).Run") {
		t.Fatalf("output should include receiver-qualified method name, got:\n%s", first.Output)
	}
	if first.Bundle.Identity.Canonical != stableKey {
		t.Fatalf("canonical = %q, want stable key %q", first.Bundle.Identity.Canonical, stableKey)
	}
	if second.Bundle.Identity.Canonical != first.Bundle.Identity.Canonical {
		t.Fatalf("canonical changed across line move: %q vs %q", second.Bundle.Identity.Canonical, first.Bundle.Identity.Canonical)
	}
}

func TestStructuredGoImpactSemanticProductionMatchesImplementationLimitAndReadOrder(t *testing.T) {
	plan := impactplan.PlanForRisk(impactplan.RiskHigh)
	totalImplementations := plan.ImplementationLimit + 3
	resolved := buildStructuredGoProductionTestResult(t, goStructuredProductionImplementationFixture(totalImplementations), plan)

	implementationSection := symbolBundleSectionByKind(resolved.Bundle, SemanticReferenceSectionKindImplementations)
	if implementationSection == nil {
		t.Fatal("implementations section = nil, want section")
	}
	if len(implementationSection.Items) != plan.ImplementationLimit {
		t.Fatalf("implementation item count = %d, want plan limit %d", len(implementationSection.Items), plan.ImplementationLimit)
	}
	if implementationSection.Total != totalImplementations || !implementationSection.More {
		t.Fatalf("implementation summary = total:%d more:%v, want total:%d more:true", implementationSection.Total, implementationSection.More, totalImplementations)
	}
	assertGoStructuredProductionRecommendedReadKinds(t, resolved.Bundle.Impact.RecommendedReads,
		"definition",
		"callers",
		"tests",
		"implementations",
		"implementations",
		"implementations",
		"references",
	)
	assertRecommendedReadAt(t, resolved.Bundle.Impact.RecommendedReads, 3, "implementations", "pkg/impl_01.go")
	assertRecommendedReadAt(t, resolved.Bundle.Impact.RecommendedReads, 4, "implementations", "pkg/impl_02.go")
	assertRecommendedReadAt(t, resolved.Bundle.Impact.RecommendedReads, 5, "implementations", "pkg/impl_03.go")
	assertOutputContains(t, resolved.Output, "Related Implementations: 8 shown (of 11)")
	assertOutputContains(t, resolved.Output, "Omitted: implementations +3")
}

func TestStructuredGoImpactSemanticProductionCarriesGoDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		result navigation.InspectResult
		check  func(t *testing.T, diagnostics SymbolBundleDiagnostics, output string)
	}{
		{
			name:   "lsp",
			result: goStructuredProductionFunctionFixture(),
			check: func(t *testing.T, diagnostics SymbolBundleDiagnostics, output string) {
				assertDiagnosticsResolvedBy(t, diagnostics, SymbolBundleResolvedByLSP)
				assertBoolPtr(t, diagnostics.LSPAttempted, true, "LSPAttempted")
				assertBoolPtr(t, diagnostics.LSPAvailable, true, "LSPAvailable")
				assertIntPtr(t, diagnostics.RawRefCount, 4, "RawRefCount")
				assertIntPtr(t, diagnostics.AcceptedRefCount, 3, "AcceptedRefCount")
				assertOutputContains(t, output, "Diagnostics: resolved_by=lsp")
			},
		},
		{
			name: "mixed fallback reason lsp error",
			result: goStructuredProductionDiagnosticFixture(navigation.InspectReferenceDiagnostics{
				ResolvedBy:       SymbolBundleResolvedByMixed,
				LSPAttempted:     true,
				LSPAvailable:     true,
				FallbackUsed:     true,
				FallbackReason:   SymbolBundleFallbackReasonLSPError,
				RawRefCount:      6,
				AcceptedRefCount: 4,
				DroppedRefCount:  2,
			}, false, false, false),
			check: func(t *testing.T, diagnostics SymbolBundleDiagnostics, output string) {
				assertDiagnosticsResolvedBy(t, diagnostics, SymbolBundleResolvedByMixed)
				assertBoolPtr(t, diagnostics.FallbackUsed, true, "FallbackUsed")
				if diagnostics.FallbackReason != SymbolBundleFallbackReasonLSPError {
					t.Fatalf("FallbackReason = %q, want %q", diagnostics.FallbackReason, SymbolBundleFallbackReasonLSPError)
				}
				assertIntPtr(t, diagnostics.DroppedRefCount, 2, "DroppedRefCount")
				assertOutputContains(t, output, "Diagnostics: resolved_by=mixed")
				assertOutputContains(t, output, "fallback_reason=lsp_error")
			},
		},
		{
			name: "fallback",
			result: goStructuredProductionDiagnosticFixture(navigation.InspectReferenceDiagnostics{
				ResolvedBy:       SymbolBundleResolvedByFallback,
				LSPAttempted:     true,
				LSPAvailable:     false,
				FallbackUsed:     true,
				FallbackReason:   SymbolBundleFallbackReasonLSPUnavailable,
				RawRefCount:      3,
				AcceptedRefCount: 2,
				DroppedRefCount:  1,
			}, false, false, false),
			check: func(t *testing.T, diagnostics SymbolBundleDiagnostics, output string) {
				assertDiagnosticsResolvedBy(t, diagnostics, SymbolBundleResolvedByFallback)
				assertBoolPtr(t, diagnostics.FallbackUsed, true, "FallbackUsed")
				assertBoolPtr(t, diagnostics.LSPAvailable, false, "LSPAvailable")
				assertOutputContains(t, output, "Diagnostics: resolved_by=fallback")
			},
		},
		{
			name:   "truncated incomplete budget hit",
			result: goStructuredProductionDiagnosticFixture(navigation.InspectReferenceDiagnostics{}, true, true, true),
			check: func(t *testing.T, diagnostics SymbolBundleDiagnostics, output string) {
				assertBoolPtr(t, diagnostics.Truncated, true, "Truncated")
				assertBoolPtr(t, diagnostics.Incomplete, true, "Incomplete")
				assertBoolPtr(t, diagnostics.BudgetLimitHit, true, "BudgetLimitHit")
				if diagnostics.Confidence != SymbolBundleConfidenceLow {
					t.Fatalf("Confidence = %q, want low", diagnostics.Confidence)
				}
				assertOutputContains(t, output, "incomplete=true")
				assertOutputContains(t, output, "truncated=true")
				assertOutputContains(t, output, "budget_limit_hit=true")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := buildStructuredGoProductionTestResult(t, tt.result, impactplan.PlanForRisk(impactplan.RiskMedium))
			assertDiagnosticsEqualToGoSourceOfTruth(t, tt.result, resolved.Bundle)
			tt.check(t, resolved.Bundle.Diagnostics, resolved.Output)
		})
	}
}

func TestStructuredGoImpactSemanticProductionCacheHitKeepsIdentityDiagnosticsAndReads(t *testing.T) {
	clearSearchSidecarCaches()
	t.Cleanup(clearSearchSidecarCaches)

	ctx := structuredImpactSearchContext{
		Pattern:  "Build",
		Route:    searchRouteTrace{Language: "go", InitialLane: searchLaneSymbol, SymbolQuery: "Build"},
		CacheKey: "structured-go-production-cache-hit",
	}
	cache := &testSearchCache{data: make(map[string]string)}
	resolved := buildStructuredGoProductionTestResult(t, goStructuredProductionFunctionFixture(), impactplan.PlanForRisk(impactplan.RiskHigh))
	opts := SearchOptions{Pattern: "Build", Path: "/repo", ProjectMapRootPath: "/repo", InvocationCWD: "/repo"}

	first, ok := tryStructuredImpactSearchResult(cache, ctx, structuredImpactSameScope(opts), func(symbol string, scope structuredImpactScope) symbolResolveResult {
		return resolved
	})
	if !ok || first.Bundle == nil {
		t.Fatalf("first structured result ok=%v bundle=%v", ok, first.Bundle)
	}
	second, ok := tryStructuredImpactSearchResult(cache, ctx, structuredImpactSameScope(opts), func(symbol string, scope structuredImpactScope) symbolResolveResult {
		t.Fatal("resolver should not be called on cache hit")
		return symbolResolveResult{}
	})
	if !ok || second.Bundle == nil {
		t.Fatalf("cached structured result ok=%v bundle=%v", ok, second.Bundle)
	}

	if !reflect.DeepEqual(second.Bundle.Identity, first.Bundle.Identity) {
		t.Fatalf("cached identity = %+v, want %+v", second.Bundle.Identity, first.Bundle.Identity)
	}
	if !reflect.DeepEqual(second.Bundle.Diagnostics, first.Bundle.Diagnostics) {
		t.Fatalf("cached diagnostics = %+v, want %+v", second.Bundle.Diagnostics, first.Bundle.Diagnostics)
	}
	if !reflect.DeepEqual(second.Bundle.Impact.RecommendedReads, first.Bundle.Impact.RecommendedReads) {
		t.Fatalf("cached RecommendedReads = %+v, want %+v", second.Bundle.Impact.RecommendedReads, first.Bundle.Impact.RecommendedReads)
	}

	artifact := newStructuredImpactSearchArtifact(second)
	if artifact.Metadata.Diagnostics == nil {
		t.Fatal("cached metadata diagnostics = nil, want diagnostics snapshot")
	}
	assertDiagnosticsResolvedBy(t, *artifact.Metadata.Diagnostics, SymbolBundleResolvedByLSP)
	artifact.Metadata.Diagnostics.ResolvedBy = SymbolBundleResolvedByFallback
	assertDiagnosticsResolvedBy(t, second.Bundle.Diagnostics, SymbolBundleResolvedByLSP)
}
