package search

import (
	"fmt"
	"strings"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func semanticEvidenceFromGoInspectResult(query string, result navigation.InspectResult) (SemanticEvidence, bool) {
	if result.Symbol == nil {
		return SemanticEvidence{}, false
	}
	diagnostics := cloneSymbolBundleDiagnostics(goSymbolBundleDiagnostics(result))
	evidence := SemanticEvidence{
		Language: "go",
		Query:    query,
		Symbol:   result.Symbol.Name,
		Definitions: []SemanticDefinition{
			semanticDefinitionFromGoCandidate(*result.Symbol, result.Body, diagnostics),
		},
		ReferenceSections: []SemanticReferenceSection{
			semanticReferenceSectionFromCounts(SemanticReferenceSectionKindCallers, len(result.Callers), result.TotalCallers, result.MoreCallers),
			semanticReferenceSectionFromCounts(SemanticReferenceSectionKindReferences, len(result.Refs), result.TotalRefs, result.MoreRefs),
			semanticReferenceSectionFromCounts(SemanticReferenceSectionKindTests, len(result.Tests), result.TotalTests, result.MoreTests),
			semanticReferenceSectionFromCounts(SemanticReferenceSectionKindImplementations, min(len(result.Implementations), goImplementationLimit), len(result.Implementations), len(result.Implementations) > goImplementationLimit),
		},
		Diagnostics: &diagnostics,
		Source:      diagnostics.ResolvedBy,
		Confidence:  diagnostics.Confidence,
	}
	evidence.References = append(evidence.References, semanticReferencesFromGoReferences(result.Callers, SemanticReferenceKindCaller, diagnostics)...)
	evidence.References = append(evidence.References, semanticReferencesFromGoReferences(result.Refs, SemanticReferenceKindReference, diagnostics)...)
	evidence.References = append(evidence.References, semanticReferencesFromGoTests(result.Tests, diagnostics)...)
	evidence.References = append(evidence.References, semanticReferencesFromGoImplementations(limitGoImplementationsForSemanticEvidence(result.Implementations), diagnostics)...)
	return evidence, true
}

func semanticDefinitionFromGoCandidate(candidate navigation.SymbolCandidate, body []string, diagnostics SymbolBundleDiagnostics) SemanticDefinition {
	return SemanticDefinition{
		Name:           candidate.Name,
		DisplayName:    goSymbolBundleDisplayName(candidate),
		Canonical:      canonicalGoSymbolBundleKey(candidate),
		Kind:           candidate.Kind,
		Exported:       candidate.Exported,
		Implementation: true,
		Declaration:    false,
		File:           candidate.File,
		Line:           candidate.Line,
		EndLine:        candidate.EndLine,
		Signature:      candidate.Signature,
		Body:           append([]string(nil), body...),
		RootPath:       candidate.RootPath,
		Source:         diagnostics.ResolvedBy,
		Confidence:     diagnostics.Confidence,
	}
}

func semanticReferenceSectionFromCounts(kind string, visible, total int, more bool) SemanticReferenceSection {
	if total < visible {
		total = visible
	}
	if total <= 0 {
		total = visible
	}
	if total > visible {
		more = true
	}
	return SemanticReferenceSection{
		Kind:  kind,
		Total: total,
		More:  more,
	}
}

func semanticReferencesFromGoReferences(refs []navigation.Reference, kind string, diagnostics SymbolBundleDiagnostics) []SemanticReference {
	semanticRefs := make([]SemanticReference, 0, len(refs))
	for _, ref := range refs {
		snippet := strings.TrimSpace(ref.Snippet)
		if snippet == "" && ref.Scope != "" {
			snippet = ref.Scope
		}
		semanticRefs = append(semanticRefs, SemanticReference{
			Kind:         kind,
			File:         ref.File,
			ResolvedPath: ref.ResolvedPath,
			Line:         ref.Line,
			Snippet:      snippet,
			Scope:        ref.Scope,
			IsTest:       ref.IsTest,
			Source:       diagnostics.ResolvedBy,
			Confidence:   diagnostics.Confidence,
		})
	}
	return semanticRefs
}

func semanticReferencesFromGoTests(tests []navigation.TestRef, diagnostics SymbolBundleDiagnostics) []SemanticReference {
	semanticRefs := make([]SemanticReference, 0, len(tests))
	for _, test := range tests {
		semanticRefs = append(semanticRefs, SemanticReference{
			Kind:         SemanticReferenceKindTest,
			File:         test.File,
			ResolvedPath: test.ResolvedPath,
			Line:         test.Line,
			Name:         test.Name,
			IsTest:       true,
			Source:       diagnostics.ResolvedBy,
			Confidence:   diagnostics.Confidence,
		})
	}
	return semanticRefs
}

func semanticReferencesFromGoImplementations(impls []navigation.ImplementationRef, diagnostics SymbolBundleDiagnostics) []SemanticReference {
	semanticRefs := make([]SemanticReference, 0, len(impls))
	for _, impl := range impls {
		semanticRefs = append(semanticRefs, SemanticReference{
			Kind:         SemanticReferenceKindImplementation,
			File:         impl.File,
			ResolvedPath: impl.ResolvedPath,
			Line:         impl.Line,
			Snippet:      strings.TrimSpace(impl.Name),
			Name:         impl.Name,
			Source:       diagnostics.ResolvedBy,
			Confidence:   diagnostics.Confidence,
		})
	}
	return semanticRefs
}

func limitGoImplementationsForSemanticEvidence(impls []navigation.ImplementationRef) []navigation.ImplementationRef {
	if len(impls) <= goImplementationLimit {
		return impls
	}
	return impls[:goImplementationLimit]
}

func semanticEvidenceFromJSFamilyRefs(language, symbol string, def genericSymbolDef, refs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	return semanticEvidenceFromJSFamilyRefsWithTotals(language, symbol, def, refs, refs, diagnostics)
}

func semanticEvidenceFromJSFamilyRefsWithTotals(language, symbol string, def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) (SemanticEvidence, bool) {
	return semanticEvidenceFromJSFamilyRefsWithRiskLevel(language, symbol, def, refs, totalRefs, diagnostics, "")
}

func semanticEvidenceFromJSFamilyRefsWithRiskLevel(language, symbol string, def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics, riskLevelOverride string) (SemanticEvidence, bool) {
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
	semanticRefs, sectionSummaries := semanticEvidenceFromJSFamilyReferenceGroups(def, filteredRefs, filteredTotalRefs, diagnostics)
	definitionFacts := semanticEvidenceJSFamilyDefinitionFactsForRefs(language, def, filteredRefs, filteredTotalRefs, riskLevelOverride)
	evidence := SemanticEvidence{
		Language:          strings.TrimSpace(language),
		Query:             symbol,
		Symbol:            symbol,
		Definitions:       []SemanticDefinition{semanticDefinitionFromJSFamilyDef(def, diagnostics, definitionFacts)},
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

func semanticEvidenceJSFamilyDefinitionFactsForRefs(language string, def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, riskLevelOverride string) semanticEvidenceJSFamilyDefinitionFacts {
	facts := semanticEvidenceJSFamilyDefinitionFacts{
		Implementation: semanticEvidenceJSFamilyDefinitionIsImplementation(def),
		Declaration:    semanticEvidenceJSFamilyDefinitionIsDeclaration(def),
		RiskLevel:      strings.TrimSpace(riskLevelOverride),
	}

	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript":
		impactRefs := typeScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, SearchOptions{})
		facts.Exported = typeScriptDefinitionIsExported(def, impactRefs)
		if facts.RiskLevel == "" {
			facts.RiskLevel = classifyTypeScriptImpactRisk(def, impactRefs)
		}
	case "javascript":
		impactRefs := javaScriptImpactRefsForDisplayAndTotalRefs(def, refs, totalRefs, SearchOptions{})
		facts.Exported = javaScriptDefinitionIsExported(def, impactRefs)
		if facts.RiskLevel == "" {
			facts.RiskLevel = classifyJavaScriptImpactRisk(def, impactRefs)
		}
	default:
		facts.Exported = def.Exported || jsFamilySignatureStartsWithExport(def.Signature) || jsFamilyRefsContainExport(totalRefs)
	}

	return facts
}

func semanticDefinitionFromJSFamilyDef(def genericSymbolDef, diagnostics SymbolBundleDiagnostics, facts semanticEvidenceJSFamilyDefinitionFacts) SemanticDefinition {
	return SemanticDefinition{
		Name:           def.Name,
		Kind:           def.Kind,
		Exported:       facts.Exported,
		Implementation: facts.Implementation,
		Declaration:    facts.Declaration,
		File:           def.File,
		Line:           def.Line,
		EndLine:        def.Line,
		Signature:      def.Signature,
		Body:           []string{fmt.Sprintf("%d: %s", def.Line, def.Signature)},
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

func semanticEvidenceFromJSFamilyReferenceGroups(def genericSymbolDef, refs []genericSymbolRef, totalRefs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) ([]SemanticReference, []SemanticReferenceSection) {
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
		semanticRefs = append(semanticRefs, semanticReferencesFromJSFamilyRefs(selected, diagnostics)...)
		total := len(dedupeGenericRefs(group.totalRefs))
		sectionSummaries = append(sectionSummaries, semanticReferenceSectionFromCounts(group.sectionKind, len(selected), total, total > len(selected)))
	}
	return semanticRefs, sectionSummaries
}

func semanticReferencesFromJSFamilyRefs(refs []genericSymbolRef, diagnostics SymbolBundleDiagnostics) []SemanticReference {
	semanticRefs := make([]SemanticReference, 0, len(refs))
	for _, ref := range refs {
		kind, ok := semanticReferenceKindFromJSFamilyRef(ref)
		if !ok {
			continue
		}
		semanticRefs = append(semanticRefs, SemanticReference{
			Kind:       kind,
			File:       ref.File,
			Line:       ref.Line,
			EndLine:    ref.Line,
			Snippet:    strings.TrimSpace(ref.Snippet),
			IsTest:     ref.IsTest || kind == SemanticReferenceKindTest,
			Source:     diagnostics.ResolvedBy,
			Confidence: diagnostics.Confidence,
		})
	}
	return semanticRefs
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
