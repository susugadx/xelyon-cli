package search

import "strings"

func buildJSFamilySemanticEvidenceShadowBundle(language string, symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) *SymbolBundle {
	refSet := jsFamilySemanticEvidenceRefsForLanguage(language, def, opts, refs, totalRefs)
	evidence, ok := semanticEvidenceFromJSFamilyRefsWithTotals(language, symbol, def, refSet.refs, refSet.totalRefs, diagnostics)
	if !ok {
		return nil
	}
	bundle, ok := buildSymbolBundleFromSemanticEvidence(evidence)
	if !ok {
		return nil
	}
	return bundle
}

type jsFamilySemanticEvidenceRefSet struct {
	refs      []genericSymbolRef
	totalRefs []genericSymbolRef
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
	}
}

func javaScriptSemanticEvidenceRefSet(def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef) jsFamilySemanticEvidenceRefSet {
	impactRefs := javaScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, opts)
	return jsFamilySemanticEvidenceRefSet{
		refs:      flattenJSFamilySemanticEvidenceRefs(impactRefs.imports, impactRefs.callers, impactRefs.others, impactRefs.allTests()),
		totalRefs: flattenJSFamilySemanticEvidenceRefs(impactRefs.totalImports, impactRefs.totalCallers, impactRefs.totalOthers, impactRefs.allTotalTests()),
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
