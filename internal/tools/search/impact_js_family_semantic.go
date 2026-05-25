package search

import "strings"

type jsFamilySemanticEvidenceBuilder func(language string, symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool)

func buildJSFamilySemanticEvidence(language string, symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	if totalRefs == nil {
		totalRefs = refs
	}
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript":
		return semanticEvidenceFromTypeScriptImpactRefs(symbol, def, opts, refs, totalRefs, diagnostics)
	case "javascript":
		return semanticEvidenceFromJavaScriptImpactRefs(symbol, def, opts, refs, totalRefs, diagnostics)
	default:
		return semanticEvidenceFromJSFamilyRefsWithOptionsAndTotals(language, symbol, def, opts, refs, totalRefs, diagnostics)
	}
}

func semanticEvidenceFromTypeScriptImpactRefs(symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	impactRefs := typeScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, opts)
	riskLevel := classifyTypeScriptImpactRisk(def, impactRefs)
	facts := semanticEvidenceJSFamilyDefinitionFacts{
		Exported:       typeScriptDefinitionIsExported(def, impactRefs),
		Implementation: semanticEvidenceJSFamilyDefinitionIsImplementation(def),
		Declaration:    semanticEvidenceJSFamilyDefinitionIsDeclaration(def),
		RiskLevel:      riskLevel,
	}
	groups := []jsFamilySemanticEvidenceGroup{
		{sectionKind: SemanticReferenceSectionKindImports, refs: impactRefs.imports, totalRefs: impactRefs.totalImports, limit: jsImportLimit},
		{sectionKind: SemanticReferenceSectionKindCallers, refs: impactRefs.callers, totalRefs: impactRefs.totalCallers, limit: jsCallerLimit},
		{sectionKind: SemanticReferenceSectionKindTypeRefs, refs: impactRefs.typeRefs, totalRefs: impactRefs.totalTypeRefs, limit: jsTypeRefLimit},
		{sectionKind: SemanticReferenceSectionKindReferences, refs: impactRefs.others, totalRefs: impactRefs.totalOthers, limit: genericRefLimit},
		{sectionKind: SemanticReferenceSectionKindTests, refs: impactRefs.allTests(), totalRefs: impactRefs.allTotalTests(), limit: genericTestLimit, isTest: true},
	}
	return semanticEvidenceFromJSFamilyImpactGroups(
		"typescript",
		symbol,
		def,
		opts,
		diagnostics,
		facts,
		groups,
		typeScriptImpactRecommendedReadGroups(def, impactRefs),
	)
}

func semanticEvidenceFromJavaScriptImpactRefs(symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	impactRefs := javaScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, opts)
	riskLevel := classifyJavaScriptImpactRisk(def, impactRefs)
	facts := semanticEvidenceJSFamilyDefinitionFacts{
		Exported:       javaScriptDefinitionIsExported(def, impactRefs),
		Implementation: semanticEvidenceJSFamilyDefinitionIsImplementation(def),
		Declaration:    semanticEvidenceJSFamilyDefinitionIsDeclaration(def),
		RiskLevel:      riskLevel,
	}
	groups := []jsFamilySemanticEvidenceGroup{
		{sectionKind: SemanticReferenceSectionKindImports, refs: impactRefs.imports, totalRefs: impactRefs.totalImports, limit: jsImportLimit},
		{sectionKind: SemanticReferenceSectionKindCallers, refs: impactRefs.callers, totalRefs: impactRefs.totalCallers, limit: jsCallerLimit},
		{sectionKind: SemanticReferenceSectionKindReferences, refs: impactRefs.others, totalRefs: impactRefs.totalOthers, limit: genericRefLimit},
		{sectionKind: SemanticReferenceSectionKindTests, refs: impactRefs.allTests(), totalRefs: impactRefs.allTotalTests(), limit: genericTestLimit, isTest: true},
	}
	return semanticEvidenceFromJSFamilyImpactGroups(
		"javascript",
		symbol,
		def,
		opts,
		diagnostics,
		facts,
		groups,
		javaScriptImpactRecommendedReadGroups(impactRefs),
	)
}

type jsFamilySemanticEvidenceGroup struct {
	sectionKind string
	refs        []genericSymbolRef
	totalRefs   []genericSymbolRef
	limit       int
	isTest      bool
}

func semanticEvidenceFromJSFamilyImpactGroups(language string, symbol string, def genericSymbolDef, opts SearchOptions, diagnostics SymbolBundleDiagnostics, facts semanticEvidenceJSFamilyDefinitionFacts, groups []jsFamilySemanticEvidenceGroup, readGroups []jsFamilyImpactReadGroup) (SemanticEvidence, bool) {
	if strings.TrimSpace(def.File) == "" || def.Line <= 0 {
		return SemanticEvidence{}, false
	}
	diagnostics = cloneSymbolBundleDiagnostics(diagnostics)
	normalizeSymbolBundleDiagnostics(&diagnostics)
	rootPath := jsFamilySemanticEvidenceRootPath(language, opts)
	semanticRefs, sectionSummaries := semanticEvidenceFromJSFamilyImpactReferenceGroups(def, groups, diagnostics, rootPath)
	evidence := SemanticEvidence{
		Language:             strings.TrimSpace(language),
		Query:                symbol,
		Symbol:               symbol,
		Definitions:          []SemanticDefinition{semanticDefinitionFromJSFamilyDef(def, diagnostics, facts, rootPath)},
		References:           semanticRefs,
		ReferenceSections:    sectionSummaries,
		RecommendedReads:     jsFamilySemanticEvidenceRecommendedReads(def, rootPath, facts.RiskLevel, readGroups),
		RecommendedReadLimit: jsFamilyImpactRecommendedReadLimit,
		Diagnostics:          &diagnostics,
		Source:               diagnostics.ResolvedBy,
		Confidence:           diagnostics.Confidence,
		RiskLevel:            facts.RiskLevel,
	}
	if semanticDefinitionDisplayName(evidence, evidence.Definitions[0]) == "" {
		return SemanticEvidence{}, false
	}
	return evidence, true
}

func semanticEvidenceFromJSFamilyImpactReferenceGroups(def genericSymbolDef, groups []jsFamilySemanticEvidenceGroup, diagnostics SymbolBundleDiagnostics, rootPath string) ([]SemanticReference, []SemanticReferenceSection) {
	var semanticRefs []SemanticReference
	sectionSummaries := make([]SemanticReferenceSection, 0, len(groups))
	for _, group := range groups {
		selected := prioritizeGenericRefs(def, group.refs, group.limit, group.isTest)
		if len(selected) == 0 {
			continue
		}
		semanticRefs = append(semanticRefs, semanticReferencesFromJSFamilyRefs(selected, diagnostics, rootPath)...)
		total := len(dedupeGenericRefs(group.totalRefs))
		sectionSummaries = append(sectionSummaries, semanticReferenceSectionFromCounts(group.sectionKind, len(selected), total, total > len(selected)))
	}
	return semanticRefs, sectionSummaries
}

func jsFamilySemanticEvidenceRecommendedReads(def genericSymbolDef, rootPath string, riskLevel string, groups []jsFamilyImpactReadGroup) []SymbolBundleItem {
	impact := buildJSFamilyImpactMetadata(def, rootPath, riskLevel, groups)
	if impact == nil {
		return nil
	}
	return append([]SymbolBundleItem(nil), impact.RecommendedReads...)
}
