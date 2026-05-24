package search

import "strings"

type jsFamilySemanticEvidenceShadowResult struct {
	Bundle   *SymbolBundle
	Evidence *SemanticEvidence
}

func buildJSFamilySemanticEvidenceShadow(language string, symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) jsFamilySemanticEvidenceShadowResult {
	refSet := jsFamilySemanticEvidenceRefsForLanguage(language, def, opts, refs, totalRefs)
	evidence, ok := semanticEvidenceFromJSFamilyRefsWithRiskLevel(language, symbol, def, refSet.refs, refSet.totalRefs, diagnostics, refSet.riskLevel)
	if !ok {
		return jsFamilySemanticEvidenceShadowResult{}
	}
	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		return jsFamilySemanticEvidenceShadowResult{}
	}
	return jsFamilySemanticEvidenceShadowResult{Bundle: bundle, Evidence: &evidence}
}

type jsFamilySemanticEvidenceRefSet struct {
	refs      []genericSymbolRef
	totalRefs []genericSymbolRef
	riskLevel string
}

func jsFamilySemanticEvidenceRefsForLanguage(language string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef) jsFamilySemanticEvidenceRefSet {
	if totalRefs == nil {
		totalRefs = refs
	}
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript":
		return typeScriptSemanticEvidenceRefSet(def, opts, refs, totalRefs)
	case "javascript":
		return javaScriptSemanticEvidenceRefSet(def, opts, refs, totalRefs)
	default:
		return jsFamilySemanticEvidenceRefSet{refs: refs, totalRefs: totalRefs}
	}
}

func typeScriptSemanticEvidenceRefSet(def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef) jsFamilySemanticEvidenceRefSet {
	impactRefs := typeScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, opts)
	return jsFamilySemanticEvidenceRefSet{
		refs:      flattenJSFamilySemanticEvidenceRefs(impactRefs.imports, impactRefs.callers, impactRefs.typeRefs, impactRefs.others, impactRefs.allTests()),
		totalRefs: flattenJSFamilySemanticEvidenceRefs(impactRefs.totalImports, impactRefs.totalCallers, impactRefs.totalTypeRefs, impactRefs.totalOthers, impactRefs.allTotalTests()),
		riskLevel: classifyTypeScriptImpactRisk(def, impactRefs),
	}
}

func javaScriptSemanticEvidenceRefSet(def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef) jsFamilySemanticEvidenceRefSet {
	impactRefs := javaScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, opts)
	return jsFamilySemanticEvidenceRefSet{
		refs:      flattenJSFamilySemanticEvidenceRefs(impactRefs.imports, impactRefs.callers, impactRefs.others, impactRefs.allTests()),
		totalRefs: flattenJSFamilySemanticEvidenceRefs(impactRefs.totalImports, impactRefs.totalCallers, impactRefs.totalOthers, impactRefs.allTotalTests()),
		riskLevel: classifyJavaScriptImpactRisk(def, impactRefs),
	}
}

func flattenJSFamilySemanticEvidenceRefs(groups ...[]genericSymbolRef) []genericSymbolRef {
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	flattened := make([]genericSymbolRef, 0, count)
	for _, group := range groups {
		flattened = append(flattened, group...)
	}
	return flattened
}
