package search

import (
	"fmt"
	"strings"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

func semanticEvidenceFromJSFamilyRefs(language, symbol string, def genericSymbolDef, refs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	return semanticEvidenceFromJSFamilyRefsWithOptions(language, symbol, def, SearchOptions{}, refs, diagnostics)
}

func semanticEvidenceFromJSFamilyRefsWithOptions(language, symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	return semanticEvidenceFromJSFamilyRefsWithOptionsAndTotals(language, symbol, def, opts, refs, refs, diagnostics)
}

func semanticEvidenceFromJSFamilyRefsWithOptionsAndTotals(language, symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	return semanticEvidenceFromJSFamilyRefsWithOptionsAndRiskLevel(language, symbol, def, opts, refs, totalRefs, diagnostics, "")
}

func semanticEvidenceFromJSFamilyRefsWithOptionsAndRiskLevel(language, symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics, riskLevelOverride string) (SemanticEvidence, bool) {
	if strings.TrimSpace(def.File) == "" || def.Line <= 0 {
		return SemanticEvidence{}, false
	}
	diagnostics = cloneSymbolBundleDiagnostics(diagnostics)
	normalizeSymbolBundleDiagnostics(&diagnostics)
	if totalRefs == nil {
		totalRefs = refs
	}
	filteredRefs := filterGenericRefs(refs, def)
	filteredTotalRefs := filterGenericRefs(totalRefs, def)
	rootPath := jsFamilySemanticEvidenceRootPath(language, opts)
	semanticRefs, sectionSummaries := semanticEvidenceFromJSFamilyReferenceGroups(def, filteredRefs, filteredTotalRefs, diagnostics, rootPath)
	definitionFacts := semanticEvidenceJSFamilyDefinitionFactsForRefs(language, def, opts, filteredRefs, filteredTotalRefs, riskLevelOverride)
	evidence := SemanticEvidence{
		Language:          strings.TrimSpace(language),
		Query:             symbol,
		Symbol:            symbol,
		Definitions:       []SemanticDefinition{semanticDefinitionFromJSFamilyDef(def, diagnostics, definitionFacts, rootPath)},
		References:        semanticRefs,
		ReferenceSections: sectionSummaries,
		Diagnostics:       &diagnostics,
		Source:            diagnostics.ResolvedBy,
		Confidence:        diagnostics.Confidence,
		RiskLevel:         definitionFacts.RiskLevel,
	}
	if semanticDefinitionDisplayName(evidence, evidence.Definitions[0]) == "" {
		return SemanticEvidence{}, false
	}
	return evidence, true
}

type semanticEvidenceJSFamilyDefinitionFacts struct {
	Exported       bool
	Implementation bool
	Declaration    bool
	RiskLevel      string
}

func semanticEvidenceJSFamilyDefinitionFactsForRefs(language string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef, riskLevelOverride string) semanticEvidenceJSFamilyDefinitionFacts {
	facts := semanticEvidenceJSFamilyDefinitionFacts{
		Implementation: semanticEvidenceJSFamilyDefinitionIsImplementation(def),
		Declaration:    semanticEvidenceJSFamilyDefinitionIsDeclaration(def),
		RiskLevel:      strings.TrimSpace(riskLevelOverride),
	}

	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript":
		impactRefs := typeScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, opts)
		facts.Exported = typeScriptDefinitionIsExported(def, impactRefs)
		if facts.RiskLevel == "" {
			facts.RiskLevel = classifyTypeScriptImpactRisk(def, impactRefs)
		}
	case "javascript":
		impactRefs := javaScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, opts)
		facts.Exported = javaScriptDefinitionIsExported(def, impactRefs)
		if facts.RiskLevel == "" {
			facts.RiskLevel = classifyJavaScriptImpactRisk(def, impactRefs)
		}
	default:
		facts.Exported = def.Exported || jsFamilySignatureStartsWithExport(def.Signature) || jsFamilyRefsContainExport(totalRefs)
	}

	return facts
}

func semanticDefinitionFromJSFamilyDef(def genericSymbolDef, diagnostics SymbolBundleDiagnostics, facts semanticEvidenceJSFamilyDefinitionFacts, rootPath string) SemanticDefinition {
	return SemanticDefinition{
		Name:           def.Name,
		Kind:           def.Kind,
		Exported:       facts.Exported,
		Implementation: facts.Implementation,
		Declaration:    facts.Declaration,
		File:           def.File,
		ResolvedPath:   jsFamilySemanticEvidenceResolvedPath(def.File, rootPath),
		Line:           def.Line,
		EndLine:        def.Line,
		Signature:      def.Signature,
		Body:           []string{fmt.Sprintf("%d: %s", def.Line, def.Signature)},
		RootPath:       rootPath,
		Source:         diagnostics.ResolvedBy,
		Confidence:     diagnostics.Confidence,
	}
}

func semanticEvidenceJSFamilyDefinitionIsImplementation(def genericSymbolDef) bool {
	return isTypeScriptImplementationFilePath(def.File) || isJavaScriptSourceFilePath(def.File)
}

func semanticEvidenceJSFamilyDefinitionIsDeclaration(def genericSymbolDef) bool {
	return isTypeScriptDeclarationFilePath(def.File)
}

func semanticEvidenceFromJSFamilyReferenceGroups(def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics, rootPath string) ([]SemanticReference, []SemanticReferenceSection) {
	classifiedRefs := classifyJSFamilySymbolRefsFromAST(refs)
	classifiedTotalRefs := classifyJSFamilySymbolRefsFromAST(totalRefs)
	groups := []struct {
		sectionKind string
		refs        []genericSymbolRef
		totalRefs   []genericSymbolRef
		limit       int
		isTest      bool
	}{
		{sectionKind: SemanticReferenceSectionKindImports, refs: classifiedRefs.imports, totalRefs: classifiedTotalRefs.imports, limit: jsImportLimit},
		{sectionKind: SemanticReferenceSectionKindCallers, refs: classifiedRefs.callers, totalRefs: classifiedTotalRefs.callers, limit: jsCallerLimit},
		{sectionKind: SemanticReferenceSectionKindTypeRefs, refs: classifiedRefs.typeRefs, totalRefs: classifiedTotalRefs.typeRefs, limit: jsTypeRefLimit},
		{sectionKind: SemanticReferenceSectionKindReferences, refs: classifiedRefs.others, totalRefs: classifiedTotalRefs.others, limit: genericRefLimit},
		{sectionKind: SemanticReferenceSectionKindTests, refs: classifiedRefs.tests, totalRefs: classifiedTotalRefs.tests, limit: genericTestLimit, isTest: true},
	}

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

func semanticReferencesFromJSFamilyRefs(refs []genericSymbolRef, diagnostics SymbolBundleDiagnostics, rootPath string) []SemanticReference {
	semanticRefs := make([]SemanticReference, 0, len(refs))
	for _, ref := range refs {
		kind, ok := semanticReferenceKindFromJSFamilyRef(ref)
		if !ok {
			continue
		}
		semanticRefs = append(semanticRefs, SemanticReference{
			Kind:         kind,
			File:         ref.File,
			ResolvedPath: jsFamilySemanticEvidenceResolvedPath(ref.File, rootPath),
			Line:         ref.Line,
			EndLine:      ref.Line,
			Snippet:      strings.TrimSpace(ref.Snippet),
			IsTest:       ref.IsTest || kind == SemanticReferenceKindTest,
			Source:       diagnostics.ResolvedBy,
			Confidence:   diagnostics.Confidence,
		})
	}
	return semanticRefs
}

func jsFamilySemanticEvidenceRootPath(language string, opts SearchOptions) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript":
		return structuredTypeScriptImpactFileRoot(opts)
	case "javascript":
		return structuredJavaScriptImpactFileRoot(opts)
	default:
		return invocationCWDOrGetwd(opts)
	}
}

func jsFamilySemanticEvidenceResolvedPath(file string, rootPath string) string {
	if strings.TrimSpace(rootPath) == "" {
		return ""
	}
	return absoluteAffectedFilePathWithBase(file, rootPath)
}

func semanticReferenceKindFromJSFamilyRef(ref genericSymbolRef) (string, bool) {
	if !jsFamilyReferenceClassVisible(ref.Class) {
		return "", false
	}
	if ref.IsTest {
		return SemanticReferenceKindTest, true
	}
	switch ref.Class {
	case codeast.ClassCall:
		return SemanticReferenceKindCaller, true
	case codeast.ClassImport:
		return SemanticReferenceKindImport, true
	case jsast.ClassExport:
		return SemanticReferenceKindExport, true
	case jsast.ClassTypeRef:
		return SemanticReferenceKindTypeRef, true
	default:
		return SemanticReferenceKindReference, true
	}
}
