package search

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func TestStructuredGoImpactSemanticShadowMatchesFunctionBundle(t *testing.T) {
	plan := impactplan.PlanForRisk(impactplan.RiskHigh)
	resolved := buildStructuredGoShadowTestResult(t, goSemanticShadowFunctionFixture(), plan)

	assertGoSemanticShadowBundleEquivalent(t, resolved.Bundle, resolved.SemanticShadowBundle,
		SemanticReferenceSectionKindCallers,
		SemanticReferenceSectionKindReferences,
		SemanticReferenceSectionKindTests,
		SemanticReferenceSectionKindImplementations,
	)
	assertGoSemanticShadowEvidenceMatchesProductionImpact(t, resolved.SemanticShadowEvidence, resolved.Bundle.Impact)
}

func TestStructuredGoImpactSemanticShadowPreservesMethodStableIdentity(t *testing.T) {
	stableKey := stableGoSymbolBundleKey("pkg", "Agent", "Run", "method", "func (a *Agent) Run() error")
	plan := impactplan.PlanForRisk(impactplan.RiskMedium)
	first := buildStructuredGoShadowTestResult(t, semanticEvidenceGoRunInspectFixture(stableKey), plan)

	moved := semanticEvidenceGoRunInspectFixture(stableKey)
	movedSymbol := *moved.Symbol
	movedSymbol.Line = 40
	movedSymbol.EndLine = 42
	moved.Symbol = &movedSymbol
	second := buildStructuredGoShadowTestResult(t, moved, plan)

	if first.Bundle.Identity.DisplayName != first.SemanticShadowBundle.Identity.DisplayName {
		t.Fatalf("shadow display name = %q, want production %q", first.SemanticShadowBundle.Identity.DisplayName, first.Bundle.Identity.DisplayName)
	}
	if first.SemanticShadowBundle.Identity.DisplayName != "(*Agent).Run" {
		t.Fatalf("shadow display name = %q, want (*Agent).Run", first.SemanticShadowBundle.Identity.DisplayName)
	}
	if first.Bundle.Identity.Canonical != first.SemanticShadowBundle.Identity.Canonical {
		t.Fatalf("shadow canonical = %q, want production %q", first.SemanticShadowBundle.Identity.Canonical, first.Bundle.Identity.Canonical)
	}
	if second.SemanticShadowBundle.Identity.Canonical != first.SemanticShadowBundle.Identity.Canonical {
		t.Fatalf("shadow canonical changed across line move: %q vs %q", second.SemanticShadowBundle.Identity.Canonical, first.SemanticShadowBundle.Identity.Canonical)
	}
}

func TestStructuredGoImpactSemanticShadowMatchesImplementationLimitAndReadOrder(t *testing.T) {
	plan := impactplan.PlanForRisk(impactplan.RiskHigh)
	totalImplementations := plan.ImplementationLimit + 3
	resolved := buildStructuredGoShadowTestResult(t, goSemanticShadowImplementationFixture(totalImplementations), plan)

	assertGoSemanticShadowSectionEquivalent(t, resolved.Bundle, resolved.SemanticShadowBundle, SemanticReferenceSectionKindImplementations)
	implementationSection := symbolBundleSectionByKind(resolved.SemanticShadowBundle, SemanticReferenceSectionKindImplementations)
	if implementationSection == nil {
		t.Fatal("shadow implementations section = nil, want section")
	}
	if len(implementationSection.Items) != plan.ImplementationLimit {
		t.Fatalf("shadow implementation item count = %d, want plan limit %d", len(implementationSection.Items), plan.ImplementationLimit)
	}
	if implementationSection.Total != totalImplementations || !implementationSection.More {
		t.Fatalf("shadow implementation summary = total:%d more:%v, want total:%d more:true", implementationSection.Total, implementationSection.More, totalImplementations)
	}
	assertGoSemanticShadowRecommendedReadOrder(t, resolved.Bundle, resolved.SemanticShadowBundle)
	assertGoSemanticShadowRecommendedReadKinds(t, resolved.SemanticShadowBundle.Impact.RecommendedReads,
		"definition",
		"callers",
		"tests",
		"implementations",
		"implementations",
		"implementations",
		"references",
	)
}

func TestStructuredGoImpactSemanticShadowCarriesGoDiagnostics(t *testing.T) {
	tests := []struct {
		name   string
		result navigation.InspectResult
	}{
		{
			name:   "lsp",
			result: goSemanticShadowFunctionFixture(),
		},
		{
			name: "mixed fallback reason lsp error",
			result: goSemanticShadowDiagnosticFixture(navigation.InspectReferenceDiagnostics{
				ResolvedBy:       SymbolBundleResolvedByMixed,
				LSPAttempted:     true,
				LSPAvailable:     true,
				FallbackUsed:     true,
				FallbackReason:   SymbolBundleFallbackReasonLSPError,
				RawRefCount:      6,
				AcceptedRefCount: 4,
				DroppedRefCount:  2,
			}, false, false, false),
		},
		{
			name: "fallback",
			result: goSemanticShadowDiagnosticFixture(navigation.InspectReferenceDiagnostics{
				ResolvedBy:       SymbolBundleResolvedByFallback,
				LSPAttempted:     true,
				LSPAvailable:     false,
				FallbackUsed:     true,
				FallbackReason:   SymbolBundleFallbackReasonLSPUnavailable,
				RawRefCount:      3,
				AcceptedRefCount: 2,
				DroppedRefCount:  1,
			}, false, false, false),
		},
		{
			name:   "truncated incomplete budget hit",
			result: goSemanticShadowDiagnosticFixture(navigation.InspectReferenceDiagnostics{}, true, true, true),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved := buildStructuredGoShadowTestResult(t, tt.result, impactplan.PlanForRisk(impactplan.RiskMedium))
			assertGoSemanticShadowDiagnosticsEquivalent(t, resolved)
		})
	}
}
