package search

import "strings"

type semanticEvidenceSectionGroup struct {
	kind  string
	title string
	items []SymbolBundleItem
}

var semanticEvidenceSectionOrder = []semanticEvidenceSectionGroup{
	{kind: SemanticReferenceSectionKindCallers, title: "Callers"},
	{kind: SemanticReferenceSectionKindTests, title: "Related Tests"},
	{kind: SemanticReferenceSectionKindImplementations, title: "Related Implementations"},
	{kind: SemanticReferenceSectionKindImports, title: "Imports"},
	{kind: SemanticReferenceSectionKindReferences, title: "References"},
	{kind: SemanticReferenceSectionKindTypeRefs, title: "Type References"},
}

var jsFamilySemanticEvidenceSectionOrder = []semanticEvidenceSectionGroup{
	{kind: SemanticReferenceSectionKindImports, title: "Imports"},
	{kind: SemanticReferenceSectionKindCallers, title: "Callers"},
	{kind: SemanticReferenceSectionKindTypeRefs, title: "Type References"},
	{kind: SemanticReferenceSectionKindReferences, title: "References"},
	{kind: SemanticReferenceSectionKindTests, title: "Related Tests"},
}

func semanticEvidenceSections(evidence SemanticEvidence) []SymbolBundleSection {
	ordered := cloneSemanticEvidenceSectionOrder(evidence.Language)
	byKind := make(map[string]*semanticEvidenceSectionGroup, len(ordered))
	for i := range ordered {
		byKind[ordered[i].kind] = &ordered[i]
	}

	for _, ref := range evidence.References {
		item, ok := semanticReferenceItem(ref)
		if !ok {
			continue
		}
		group := byKind[item.Kind]
		if group == nil {
			continue
		}
		group.items = append(group.items, item)
	}

	summaries := semanticEvidenceSectionSummaries(evidence.ReferenceSections)
	sections := make([]SymbolBundleSection, 0, len(ordered))
	for _, group := range ordered {
		if len(group.items) == 0 {
			continue
		}
		total, more := semanticEvidenceSectionTotalAndMore(group.kind, len(group.items), summaries)
		sections = append(sections, SymbolBundleSection{
			Kind:  group.kind,
			Title: group.title,
			Items: group.items,
			Total: total,
			More:  more,
		})
	}
	return sections
}

func cloneSemanticEvidenceSectionOrder(language string) []semanticEvidenceSectionGroup {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "typescript", "javascript":
		return append([]semanticEvidenceSectionGroup(nil), jsFamilySemanticEvidenceSectionOrder...)
	}
	return append([]semanticEvidenceSectionGroup(nil), semanticEvidenceSectionOrder...)
}

type semanticEvidenceSectionSummary struct {
	total int
	more  bool
}

func semanticEvidenceSectionSummaries(sections []SemanticReferenceSection) map[string]semanticEvidenceSectionSummary {
	summaries := make(map[string]semanticEvidenceSectionSummary, len(sections))
	for _, section := range sections {
		kind := strings.TrimSpace(section.Kind)
		if kind == "" {
			continue
		}
		summary := summaries[kind]
		if section.Total > summary.total {
			summary.total = section.Total
		}
		summary.more = summary.more || section.More
		summaries[kind] = summary
	}
	return summaries
}

func semanticEvidenceSectionTotalAndMore(kind string, visible int, summaries map[string]semanticEvidenceSectionSummary) (int, bool) {
	total := visible
	more := false
	if summary, ok := summaries[kind]; ok {
		if summary.total > total {
			total = summary.total
		}
		more = summary.more || total > visible
	}
	return total, more
}

func semanticReferenceItem(ref SemanticReference) (SymbolBundleItem, bool) {
	itemKind, ok := semanticReferenceItemKind(ref.Kind)
	if !ok {
		return SymbolBundleItem{}, false
	}
	if strings.TrimSpace(ref.File) == "" || ref.Line <= 0 {
		return SymbolBundleItem{}, false
	}
	isTest := ref.IsTest || ref.Kind == SemanticReferenceKindTest || ref.Kind == SemanticReferenceKindNearbyTest
	return SymbolBundleItem{
		Kind:         itemKind,
		File:         ref.File,
		ResolvedPath: ref.ResolvedPath,
		Line:         ref.Line,
		EndLine:      ref.EndLine,
		Snippet:      strings.TrimSpace(ref.Snippet),
		Scope:        ref.Scope,
		Name:         ref.Name,
		IsTest:       isTest,
	}, true
}

func semanticReferenceItemKind(kind string) (string, bool) {
	switch kind {
	case SemanticReferenceKindCaller:
		return SemanticReferenceSectionKindCallers, true
	case SemanticReferenceKindTest, SemanticReferenceKindNearbyTest:
		return SemanticReferenceSectionKindTests, true
	case SemanticReferenceKindImplementation:
		return SemanticReferenceSectionKindImplementations, true
	case SemanticReferenceKindImport, SemanticReferenceKindExport:
		return SemanticReferenceSectionKindImports, true
	case SemanticReferenceKindReference:
		return SemanticReferenceSectionKindReferences, true
	case SemanticReferenceKindTypeRef:
		return SemanticReferenceSectionKindTypeRefs, true
	default:
		return "", false
	}
}
