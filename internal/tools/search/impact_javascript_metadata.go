package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
)

func javaScriptImpactRecommendedReadGroups(refs javaScriptImpactRefs) []jsFamilyImpactReadGroup {
	return []jsFamilyImpactReadGroup{
		{kind: "callers", refs: refs.callers, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "tests", refs: refs.directTests, limit: jsFamilyImpactRecommendedReadPerGroupLimit, isTest: true},
		{kind: "imports", refs: refs.imports, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "references", refs: refs.others, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "tests", refs: refs.nearbyTests, limit: jsFamilyImpactRecommendedReadPerGroupLimit, isTest: true},
	}
}

func classifyJavaScriptImpactRisk(def genericSymbolDef, refs javaScriptImpactRefs) string {
	nonTestRefCount := len(dedupeGenericRefs(append(append(append([]genericSymbolRef(nil), refs.totalImportsForRisk()...), refs.totalCallersForRisk()...), refs.totalOthersForRisk()...)))
	hasTests := len(dedupeGenericRefs(refs.allTotalTests())) > 0
	exported := javaScriptDefinitionIsExported(def, refs)

	switch {
	case !hasTests && nonTestRefCount >= jsFamilyImpactHighNonTestReferenceThreshold:
		return impactplan.RiskHigh
	case exported && !hasTests && nonTestRefCount >= jsFamilyImpactMediumNonTestReferenceThreshold:
		return impactplan.RiskHigh
	case exported || !hasTests || nonTestRefCount >= jsFamilyImpactMediumNonTestReferenceThreshold:
		return impactplan.RiskMedium
	default:
		return impactplan.RiskLow
	}
}

func javaScriptDefinitionIsExported(def genericSymbolDef, refs javaScriptImpactRefs) bool {
	if def.Exported {
		return true
	}
	signature := strings.TrimSpace(def.Signature)
	if jsFamilySignatureStartsWithExport(signature) {
		return true
	}
	if strings.HasPrefix(signature, "module.exports") || strings.HasPrefix(signature, "exports.") || strings.HasPrefix(signature, "exports[") {
		return true
	}
	return jsFamilyRefsContainExport(refs.totalImportsForRisk())
}
