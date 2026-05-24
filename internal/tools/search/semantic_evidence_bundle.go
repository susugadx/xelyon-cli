package search

import (
	"fmt"
	"strings"
)

func buildSymbolBundleFromSemanticEvidence(evidence SemanticEvidence) (*SymbolBundle, bool) {
	definition, ok := semanticEvidencePrimaryDefinition(evidence)
	if !ok {
		return nil, false
	}

	displayName := semanticDefinitionDisplayName(evidence, definition)
	query := strings.TrimSpace(evidence.Query)
	if query == "" {
		query = displayName
	}
	endLine := semanticDefinitionEndLine(definition)
	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    strings.TrimSpace(evidence.Language),
			Query:       query,
			Canonical:   semanticDefinitionCanonical(evidence, definition, displayName),
			DisplayName: displayName,
			Kind:        definition.Kind,
			File:        definition.File,
			Line:        definition.Line,
			EndLine:     endLine,
		},
		Definition: SymbolBundleDefinition{
			File:      definition.File,
			Line:      definition.Line,
			EndLine:   endLine,
			Signature: strings.TrimSpace(definition.Signature),
			Body:      append([]string(nil), definition.Body...),
		},
		Sections:    semanticEvidenceSections(evidence),
		Diagnostics: semanticEvidenceDiagnostics(evidence),
		Debug: SymbolBundleDebug{
			Source:       "semantic-evidence",
			FileRootPath: definition.RootPath,
		},
	}

	if reads := semanticEvidenceRecommendedReads(evidence, definition); len(reads) > 0 {
		bundle.Impact = &SymbolBundleImpact{RecommendedReads: reads}
	}
	finalizeSymbolBundleDiagnostics(bundle)
	return bundle, true
}

func semanticEvidencePrimaryDefinition(evidence SemanticEvidence) (SemanticDefinition, bool) {
	if len(evidence.Definitions) == 0 {
		return SemanticDefinition{}, false
	}
	definition := evidence.Definitions[0]
	if strings.TrimSpace(definition.File) == "" || definition.Line <= 0 {
		return SemanticDefinition{}, false
	}
	if semanticDefinitionSymbolName(evidence, definition) == "" {
		return SemanticDefinition{}, false
	}
	return definition, true
}

func semanticDefinitionSymbolName(evidence SemanticEvidence, definition SemanticDefinition) string {
	for _, candidate := range []string{definition.Name, definition.Symbol, evidence.Symbol} {
		if name := strings.TrimSpace(candidate); name != "" {
			return name
		}
	}
	return ""
}

func semanticDefinitionDisplayName(evidence SemanticEvidence, definition SemanticDefinition) string {
	if name := strings.TrimSpace(definition.DisplayName); name != "" {
		return name
	}
	if name := semanticDefinitionSymbolName(evidence, definition); name != "" {
		return name
	}
	return strings.TrimSpace(evidence.Query)
}

func semanticDefinitionCanonical(evidence SemanticEvidence, definition SemanticDefinition, displayName string) string {
	if canonical := strings.TrimSpace(definition.Canonical); canonical != "" {
		return canonical
	}
	return canonicalSymbolBundleKey(strings.TrimSpace(evidence.Language), definition.File, definition.Line, displayName)
}

func semanticDefinitionEndLine(definition SemanticDefinition) int {
	if definition.EndLine > 0 {
		return definition.EndLine
	}
	return definition.Line
}

func semanticEvidenceDiagnostics(evidence SemanticEvidence) SymbolBundleDiagnostics {
	var diagnostics SymbolBundleDiagnostics
	if evidence.Diagnostics != nil {
		diagnostics = cloneSymbolBundleDiagnostics(*evidence.Diagnostics)
	}
	if diagnostics.ResolvedBy == "" && strings.TrimSpace(evidence.Source) != "" {
		diagnostics.ResolvedBy = strings.TrimSpace(evidence.Source)
	}
	if diagnostics.Confidence == "" && strings.TrimSpace(evidence.Confidence) != "" {
		diagnostics.Confidence = strings.TrimSpace(evidence.Confidence)
	}
	normalizeSymbolBundleDiagnostics(&diagnostics)
	return diagnostics
}

func semanticEvidenceRecommendedReads(evidence SemanticEvidence, definition SemanticDefinition) []SymbolBundleItem {
	reads := make([]SymbolBundleItem, 0, 1+len(evidence.References))
	seen := make(map[string]struct{}, 1+len(evidence.References))
	add := func(item SymbolBundleItem) {
		if item.File == "" || item.Line <= 0 {
			return
		}
		key := semanticEvidenceReadKey(item)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		reads = append(reads, item)
	}

	add(semanticDefinitionReadItem(evidence, definition))
	for _, kind := range []string{
		SemanticReferenceKindCaller,
		SemanticReferenceKindTest,
		SemanticReferenceKindImplementation,
		SemanticReferenceKindImport,
		SemanticReferenceKindExport,
		SemanticReferenceKindReference,
		SemanticReferenceKindTypeRef,
		SemanticReferenceKindNearbyTest,
	} {
		for _, ref := range evidence.References {
			if ref.Kind != kind {
				continue
			}
			item, ok := semanticReferenceItem(ref)
			if ok {
				add(item)
			}
		}
	}
	return reads
}

func semanticEvidenceReadKey(item SymbolBundleItem) string {
	return fmt.Sprintf("%s:%d:%s", item.File, item.Line, item.Kind)
}

func semanticDefinitionReadItem(evidence SemanticEvidence, definition SemanticDefinition) SymbolBundleItem {
	return SymbolBundleItem{
		Kind:         "definition",
		File:         definition.File,
		ResolvedPath: definition.ResolvedPath,
		Line:         definition.Line,
		EndLine:      semanticDefinitionEndLine(definition),
		Snippet:      strings.TrimSpace(definition.Signature),
		Name:         semanticDefinitionDisplayName(evidence, definition),
	}
}
