package search

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func buildGoImpactMetadata(result navigation.InspectResult, risk string) *SymbolBundleImpact {
	if result.Symbol == nil {
		return nil
	}

	impact := &SymbolBundleImpact{
		RiskLevel:        risk,
		RecommendedReads: make([]SymbolBundleItem, 0, 8),
	}

	seen := make(map[string]struct{})
	add := func(item SymbolBundleItem) {
		if item.File == "" || item.Line <= 0 {
			return
		}
		key := fmt.Sprintf("%s:%d:%s", item.File, item.Line, item.Kind)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		impact.RecommendedReads = append(impact.RecommendedReads, item)
	}

	add(SymbolBundleItem{
		Kind:    "definition",
		File:    result.Symbol.File,
		Line:    result.Symbol.Line,
		EndLine: result.Symbol.EndLine,
		Snippet: impactDefinitionSnippet(result),
		Name:    result.Symbol.Name,
	})

	for _, item := range primaryCallerReadItems(result.Callers, impactReadLimit(risk, "callers")) {
		add(item)
	}
	for _, item := range primaryTestReadItems(result.Tests, impactReadLimit(risk, "tests")) {
		add(item)
	}
	for _, item := range primaryImplementationReadItems(result.Implementations, impactReadLimit(risk, "implementations")) {
		add(item)
	}
	for _, item := range crossPackageRefReadItems(result, impactReadLimit(risk, "references")) {
		add(item)
	}

	if len(impact.RecommendedReads) == 0 {
		return nil
	}
	return impact
}

func impactReadLimit(risk, kind string) int {
	switch kind {
	case "callers":
		switch risk {
		case impactplan.RiskHigh:
			return 3
		case impactplan.RiskMedium:
			return 2
		default:
			return 1
		}
	case "tests":
		if risk == impactplan.RiskHigh {
			return 2
		}
		return 1
	case "implementations":
		switch risk {
		case impactplan.RiskHigh:
			return 3
		case impactplan.RiskMedium:
			return 2
		default:
			return 1
		}
	case "references":
		switch risk {
		case impactplan.RiskHigh:
			return 2
		case impactplan.RiskMedium:
			return 1
		default:
			return 0
		}
	default:
		return 0
	}
}
