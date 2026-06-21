package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func semanticEvidenceFromGoInspectResult(query string, result navigation.InspectResult) (SemanticEvidence, bool) {
	return semanticEvidenceFromGoInspectResultWithOptions(query, result, goSemanticEvidenceOptions{})
}

type goSemanticEvidenceOptions struct {
	riskLevel           string
	recommendedReads    []SymbolBundleItem
	implementationLimit int
}

func semanticEvidenceFromGoInspectResultWithOptions(query string, result navigation.InspectResult, opts goSemanticEvidenceOptions) (SemanticEvidence, bool) {
	if result.Symbol == nil {
		return SemanticEvidence{}, false
	}
	if opts.implementationLimit <= 0 {
		opts.implementationLimit = goImplementationLimit
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
			semanticReferenceSectionFromCounts(SemanticReferenceSectionKindImplementations, min(len(result.Implementations), opts.implementationLimit), len(result.Implementations), len(result.Implementations) > opts.implementationLimit),
		},
		RecommendedReads: append([]SymbolBundleItem(nil), opts.recommendedReads...),
		Diagnostics:      &diagnostics,
		Source:           diagnostics.ResolvedBy,
		Confidence:       diagnostics.Confidence,
		RiskLevel:        strings.TrimSpace(opts.riskLevel),
	}
	evidence.References = append(evidence.References, semanticReferencesFromGoReferences(result.Callers, SemanticReferenceKindCaller, diagnostics)...)
	evidence.References = append(evidence.References, semanticReferencesFromGoReferences(result.Refs, SemanticReferenceKindReference, diagnostics)...)
	evidence.References = append(evidence.References, semanticReferencesFromGoTests(result.Tests, diagnostics)...)
	evidence.References = append(evidence.References, semanticReferencesFromGoImplementations(limitGoImplementationsForSemanticEvidence(result.Implementations, opts.implementationLimit), diagnostics)...)
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
		Signature:      goSemanticDefinitionSignatureFromBody(body),
		Body:           append([]string(nil), body...),
		RootPath:       candidate.RootPath,
		Source:         diagnostics.ResolvedBy,
		Confidence:     diagnostics.Confidence,
	}
}

func goSemanticDefinitionSignatureFromBody(body []string) string {
	if len(body) == 0 {
		return ""
	}
	return body[0]
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

func limitGoImplementationsForSemanticEvidence(impls []navigation.ImplementationRef, limit int) []navigation.ImplementationRef {
	if limit <= 0 {
		return nil
	}
	if len(impls) <= limit {
		return impls
	}
	return impls[:limit]
}
