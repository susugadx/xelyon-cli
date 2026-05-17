package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
)

func buildJavaScriptImpactBundle(symbol string, def genericSymbolDef, opts SearchOptions, refs javaScriptImpactRefs) *SymbolBundle {
	rootPath := structuredJavaScriptImpactFileRoot(opts)
	impact := buildJavaScriptImpactMetadata(def, refs, rootPath)
	if impact == nil || len(impact.RecommendedReads) == 0 {
		return nil
	}

	bundle := newJSFamilyImpactBundle(jsFamilyImpactBundleSpec{
		language:    "javascript",
		debugSource: "javascript-impact-structured",
		symbol:      symbol,
		def:         def,
		rootPath:    rootPath,
		impact:      impact,
	})

	appendJSFamilyImpactSection(bundle, def, "imports", "Imports", refs.imports, jsImportLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "callers", "Callers", refs.callers, jsCallerLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "references", "References", refs.others, genericRefLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "tests", "Related Tests", refs.allTests(), genericTestLimit, true, rootPath, symbol)

	return bundle
}

func buildJavaScriptImpactMetadata(def genericSymbolDef, refs javaScriptImpactRefs, rootPath string) *SymbolBundleImpact {
	return buildJSFamilyImpactMetadata(def, rootPath, classifyJavaScriptImpactRisk(def, refs), javaScriptImpactRecommendedReadGroups(refs))
}

func javaScriptImpactRecommendedReadGroups(refs javaScriptImpactRefs) []jsFamilyImpactReadGroup {
	return []jsFamilyImpactReadGroup{
		{kind: "callers", refs: refs.callers, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "tests", refs: refs.directTests, limit: jsFamilyImpactRecommendedReadPerGroupLimit, isTest: true},
		{kind: "imports", refs: refs.imports, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "tests", refs: refs.nearbyTests, limit: jsFamilyImpactRecommendedReadPerGroupLimit, isTest: true},
	}
}

func classifyJavaScriptImpactRisk(def genericSymbolDef, refs javaScriptImpactRefs) string {
	nonTestRefCount := len(dedupeGenericRefs(append(append(append([]genericSymbolRef(nil), refs.imports...), refs.callers...), refs.others...)))
	hasTests := len(dedupeGenericRefs(refs.allTests())) > 0
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
	return jsFamilyRefsContainExport(refs.imports)
}
