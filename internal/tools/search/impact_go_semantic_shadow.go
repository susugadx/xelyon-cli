package search

import (
	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type goSemanticShadowResult struct {
	Bundle   *SymbolBundle
	Evidence *SemanticEvidence
}

func buildGoSemanticShadowResult(symbol string, result navigation.InspectResult, production *SymbolBundle, plan impactplan.Plan) (goSemanticShadowResult, bool) {
	if production == nil {
		return goSemanticShadowResult{}, false
	}

	var reads []SymbolBundleItem
	if production.Impact != nil {
		reads = production.Impact.RecommendedReads
	}
	evidence, ok := semanticEvidenceFromGoInspectResultWithOptions(symbol, result, goSemanticEvidenceOptions{
		riskLevel:           plan.RiskLevel,
		recommendedReads:    reads,
		implementationLimit: plan.ImplementationLimit,
	})
	if !ok {
		return goSemanticShadowResult{}, false
	}
	applyProductionBundleToGoSemanticShadowEvidence(&evidence, production)

	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		return goSemanticShadowResult{Evidence: &evidence}, false
	}
	return goSemanticShadowResult{Bundle: bundle, Evidence: &evidence}, true
}

func applyProductionBundleToGoSemanticShadowEvidence(evidence *SemanticEvidence, production *SymbolBundle) {
	if evidence == nil || production == nil || len(evidence.Definitions) == 0 {
		return
	}

	definition := &evidence.Definitions[0]
	definition.DisplayName = production.Identity.DisplayName
	definition.Canonical = production.Identity.Canonical
	definition.Kind = production.Identity.Kind
	definition.File = production.Definition.File
	definition.Line = production.Definition.Line
	definition.EndLine = production.Definition.EndLine
	definition.Signature = production.Definition.Signature
	definition.Body = append([]string(nil), production.Definition.Body...)
	definition.RootPath = production.Debug.FileRootPath
}
