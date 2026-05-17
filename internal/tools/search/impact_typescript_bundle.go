package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/impactplan"
)

func buildTypeScriptImpactBundle(symbol string, def genericSymbolDef, opts SearchOptions, refs typeScriptImpactRefs) *SymbolBundle {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	impact := buildTypeScriptImpactMetadata(def, refs, rootPath)
	if impact == nil || len(impact.RecommendedReads) == 0 {
		return nil
	}

	bundle := newJSFamilyImpactBundle(jsFamilyImpactBundleSpec{
		language:    "typescript",
		debugSource: typeScriptImpactDebugSource(def),
		symbol:      symbol,
		def:         def,
		rootPath:    rootPath,
		impact:      impact,
	})

	appendJSFamilyImpactSection(bundle, def, "imports", "Imports", refs.imports, jsImportLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "callers", "Callers", refs.callers, jsCallerLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "type_refs", "Type References", refs.typeRefs, jsTypeRefLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "references", "References", refs.others, genericRefLimit, false, rootPath, symbol)
	appendJSFamilyImpactSection(bundle, def, "tests", "Related Tests", refs.allTests(), genericTestLimit, true, rootPath, symbol)

	return bundle
}

func typeScriptImpactDebugSource(def genericSymbolDef) string {
	if isTypeScriptDeclarationFilePath(def.File) {
		return "typescript-impact-structured-declaration"
	}
	return "typescript-impact-structured"
}

func buildTypeScriptImpactMetadata(def genericSymbolDef, refs typeScriptImpactRefs, rootPath string) *SymbolBundleImpact {
	return buildJSFamilyImpactMetadata(def, rootPath, classifyTypeScriptImpactRisk(def, refs), typeScriptImpactRecommendedReadGroups(def, refs))
}

func typeScriptImpactRecommendedReadGroups(def genericSymbolDef, refs typeScriptImpactRefs) []jsFamilyImpactReadGroup {
	if isTypeScriptDeclarationFilePath(def.File) {
		return typeScriptDeclarationImpactRecommendedReadGroups(refs)
	}

	importReferenceGroups := []jsFamilyImpactReadGroup{
		{kind: "imports", refs: refs.imports, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "references", refs: refs.others, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
	}
	callerGroup := jsFamilyImpactReadGroup{kind: "callers", refs: refs.callers, limit: jsFamilyImpactRecommendedReadPerGroupLimit}
	typeRefGroup := jsFamilyImpactReadGroup{kind: "type_refs", refs: refs.typeRefs, limit: jsFamilyImpactRecommendedReadPerGroupLimit}
	directTestGroup := jsFamilyImpactReadGroup{kind: "tests", refs: refs.directTests, limit: jsFamilyImpactRecommendedReadPerGroupLimit, isTest: true}
	nearbyTestGroup := jsFamilyImpactReadGroup{kind: "tests", refs: refs.nearbyTests, limit: jsFamilyImpactRecommendedReadPerGroupLimit, isTest: true}

	if typeScriptImpactPrefersTypeRefs(def.Kind) {
		groups := []jsFamilyImpactReadGroup{typeRefGroup, directTestGroup}
		groups = append(groups, importReferenceGroups...)
		groups = append(groups, nearbyTestGroup)
		return groups
	}

	groups := []jsFamilyImpactReadGroup{callerGroup, directTestGroup}
	groups = append(groups, importReferenceGroups...)
	groups = append(groups, nearbyTestGroup)
	return groups
}

func typeScriptDeclarationImpactRecommendedReadGroups(refs typeScriptImpactRefs) []jsFamilyImpactReadGroup {
	return []jsFamilyImpactReadGroup{
		{kind: "type_refs", refs: refs.typeRefs, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "imports", refs: refs.imports, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "references", refs: refs.others, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "callers", refs: refs.callers, limit: jsFamilyImpactRecommendedReadPerGroupLimit},
		{kind: "tests", refs: refs.directTests, limit: jsFamilyImpactRecommendedReadPerGroupLimit, isTest: true},
	}
}

func typeScriptImpactPrefersTypeRefs(kind string) bool {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "interface", "type":
		return true
	default:
		return false
	}
}

func classifyTypeScriptImpactRisk(def genericSymbolDef, refs typeScriptImpactRefs) string {
	nonTestRefCount := len(dedupeGenericRefs(append(append(append(append([]genericSymbolRef(nil), refs.imports...), refs.callers...), refs.typeRefs...), refs.others...)))
	hasTests := len(dedupeGenericRefs(refs.allTests())) > 0
	exported := typeScriptDefinitionIsExported(def, refs)
	primaryRefCount := nonTestRefCount
	if typeScriptImpactPrefersTypeRefs(def.Kind) {
		primaryRefCount = len(dedupeGenericRefs(refs.typeRefs))
	}

	switch {
	case !hasTests && nonTestRefCount >= jsFamilyImpactHighNonTestReferenceThreshold:
		return impactplan.RiskHigh
	case exported && !hasTests && primaryRefCount >= jsFamilyImpactMediumNonTestReferenceThreshold:
		return impactplan.RiskHigh
	case exported || !hasTests || nonTestRefCount >= jsFamilyImpactMediumNonTestReferenceThreshold || primaryRefCount >= jsFamilyImpactMediumNonTestReferenceThreshold:
		return impactplan.RiskMedium
	default:
		return impactplan.RiskLow
	}
}

func typeScriptDefinitionIsExported(def genericSymbolDef, refs typeScriptImpactRefs) bool {
	if def.Exported {
		return true
	}
	if jsFamilySignatureStartsWithExport(def.Signature) {
		return true
	}
	return jsFamilyRefsContainExport(refs.imports)
}
