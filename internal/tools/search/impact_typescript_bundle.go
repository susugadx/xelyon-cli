package search

import (
	"fmt"
	"strings"
)

const (
	typeScriptImpactRecommendedReadLimit          = 5
	typeScriptImpactRecommendedReadPerGroupLimit  = 1
	typeScriptImpactHighNonTestReferenceThreshold = 8
	typeScriptImpactMediumReferenceThreshold      = 4
)

func buildTypeScriptImpactBundle(symbol string, def genericSymbolDef, opts SearchOptions, refs typeScriptImpactRefs) *SymbolBundle {
	rootPath := structuredTypeScriptImpactFileRoot(opts)
	impact := buildTypeScriptImpactMetadata(def, refs, rootPath)
	if impact == nil || len(impact.RecommendedReads) == 0 {
		return nil
	}

	displayName := def.Name
	if displayName == "" {
		displayName = symbol
	}
	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "typescript",
			Query:       symbol,
			Canonical:   canonicalSymbolBundleKey("typescript", def.File, def.Line, displayName),
			DisplayName: displayName,
			Kind:        def.Kind,
			File:        def.File,
			Line:        def.Line,
			EndLine:     def.Line,
		},
		Definition: SymbolBundleDefinition{
			File:      def.File,
			Line:      def.Line,
			EndLine:   def.Line,
			Signature: def.Signature,
			Body:      []string{fmt.Sprintf("%d: %s", def.Line, def.Signature)},
		},
		Impact: impact,
		Debug: SymbolBundleDebug{
			Source:       typeScriptImpactDebugSource(def),
			FileRootPath: rootPath,
		},
	}

	appendTypeScriptImpactSection(bundle, def, "imports", "Imports", refs.imports, jsImportLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "callers", "Callers", refs.callers, jsCallerLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "type_refs", "Type References", refs.typeRefs, jsTypeRefLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "references", "References", refs.others, genericRefLimit, false, rootPath, symbol)
	appendTypeScriptImpactSection(bundle, def, "tests", "Related Tests", refs.allTests(), genericTestLimit, true, rootPath, symbol)

	return bundle
}

func typeScriptImpactDebugSource(def genericSymbolDef) string {
	if isTypeScriptDeclarationFilePath(def.File) {
		return "typescript-impact-structured-declaration"
	}
	return "typescript-impact-structured"
}

func appendTypeScriptImpactSection(bundle *SymbolBundle, def genericSymbolDef, kind, title string, refs []genericSymbolRef, limit int, isTest bool, rootPath string, symbol string) {
	items := typeScriptImpactItemsFromRefs(def, refs, kind, limit, isTest, rootPath, symbol)
	if len(items) == 0 {
		return
	}

	total := len(dedupeGenericRefs(refs))
	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  kind,
		Title: title,
		Items: items,
		Total: total,
		More:  total > len(items),
	})
}

func buildTypeScriptImpactMetadata(def genericSymbolDef, refs typeScriptImpactRefs, rootPath string) *SymbolBundleImpact {
	impact := &SymbolBundleImpact{
		RiskLevel:        classifyTypeScriptImpactRisk(def, refs),
		RecommendedReads: make([]SymbolBundleItem, 0, typeScriptImpactRecommendedReadLimit),
	}

	seen := make(map[string]struct{}, typeScriptImpactRecommendedReadLimit)
	add := func(item SymbolBundleItem) {
		if item.File == "" || item.Line <= 0 || len(impact.RecommendedReads) >= typeScriptImpactRecommendedReadLimit {
			return
		}
		key := structuredTypeScriptLocationKey(item.File, item.Line)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		impact.RecommendedReads = append(impact.RecommendedReads, item)
	}

	add(typeScriptImpactDefinitionItem(def, rootPath))
	for _, group := range typeScriptImpactRecommendedReadGroups(def, refs) {
		for _, item := range typeScriptImpactItemsFromRefs(def, group.refs, group.kind, group.limit, group.isTest, rootPath, def.Name) {
			add(item)
		}
	}

	if len(impact.RecommendedReads) == 0 {
		return nil
	}
	return impact
}

type typeScriptImpactReadGroup struct {
	kind   string
	refs   []genericSymbolRef
	limit  int
	isTest bool
}

func typeScriptImpactRecommendedReadGroups(def genericSymbolDef, refs typeScriptImpactRefs) []typeScriptImpactReadGroup {
	if isTypeScriptDeclarationFilePath(def.File) {
		return typeScriptDeclarationImpactRecommendedReadGroups(refs)
	}

	importReferenceGroups := []typeScriptImpactReadGroup{
		{kind: "imports", refs: refs.imports, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "references", refs: refs.others, limit: typeScriptImpactRecommendedReadPerGroupLimit},
	}
	callerGroup := typeScriptImpactReadGroup{kind: "callers", refs: refs.callers, limit: typeScriptImpactRecommendedReadPerGroupLimit}
	typeRefGroup := typeScriptImpactReadGroup{kind: "type_refs", refs: refs.typeRefs, limit: typeScriptImpactRecommendedReadPerGroupLimit}
	directTestGroup := typeScriptImpactReadGroup{kind: "tests", refs: refs.directTests, limit: typeScriptImpactRecommendedReadPerGroupLimit, isTest: true}
	nearbyTestGroup := typeScriptImpactReadGroup{kind: "tests", refs: refs.nearbyTests, limit: typeScriptImpactRecommendedReadPerGroupLimit, isTest: true}

	if typeScriptImpactPrefersTypeRefs(def.Kind) {
		groups := []typeScriptImpactReadGroup{typeRefGroup, directTestGroup}
		groups = append(groups, importReferenceGroups...)
		groups = append(groups, nearbyTestGroup)
		return groups
	}

	groups := []typeScriptImpactReadGroup{callerGroup, directTestGroup}
	groups = append(groups, importReferenceGroups...)
	groups = append(groups, nearbyTestGroup)
	return groups
}

func typeScriptDeclarationImpactRecommendedReadGroups(refs typeScriptImpactRefs) []typeScriptImpactReadGroup {
	return []typeScriptImpactReadGroup{
		{kind: "type_refs", refs: refs.typeRefs, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "imports", refs: refs.imports, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "references", refs: refs.others, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "callers", refs: refs.callers, limit: typeScriptImpactRecommendedReadPerGroupLimit},
		{kind: "tests", refs: refs.directTests, limit: typeScriptImpactRecommendedReadPerGroupLimit, isTest: true},
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

func typeScriptImpactItemsFromRefs(def genericSymbolDef, refs []genericSymbolRef, kind string, limit int, isTest bool, rootPath string, symbol string) []SymbolBundleItem {
	selected := prioritizeGenericRefs(def, refs, limit, isTest)
	if len(selected) == 0 {
		return nil
	}

	items := make([]SymbolBundleItem, 0, len(selected))
	for _, ref := range selected {
		items = append(items, typeScriptImpactItemFromRef(kind, ref, rootPath, symbol, isTest))
	}
	return items
}

func typeScriptImpactDefinitionItem(def genericSymbolDef, rootPath string) SymbolBundleItem {
	return SymbolBundleItem{
		Kind:         "definition",
		File:         def.File,
		ResolvedPath: absoluteAffectedFilePathWithBase(def.File, rootPath),
		Line:         def.Line,
		EndLine:      def.Line,
		Snippet:      strings.TrimSpace(def.Signature),
		Name:         def.Name,
	}
}

func typeScriptImpactItemFromRef(kind string, ref genericSymbolRef, rootPath string, symbol string, forceTest bool) SymbolBundleItem {
	isTest := forceTest || ref.IsTest
	name := strings.TrimSpace(symbol)
	if isTest {
		name = ""
	}
	return SymbolBundleItem{
		Kind:         kind,
		File:         ref.File,
		ResolvedPath: absoluteAffectedFilePathWithBase(ref.File, rootPath),
		Line:         ref.Line,
		EndLine:      ref.Line,
		Snippet:      strings.TrimSpace(ref.Snippet),
		Name:         name,
		IsTest:       isTest,
	}
}

func classifyTypeScriptImpactRisk(def genericSymbolDef, refs typeScriptImpactRefs) string {
	nonTestRefCount := len(dedupeGenericRefs(append(append(append(append([]genericSymbolRef(nil), refs.imports...), refs.callers...), refs.typeRefs...), refs.others...)))
	hasTests := len(dedupeGenericRefs(refs.allTests())) > 0
	exported := typeScriptDefinitionIsExported(def)
	primaryRefCount := nonTestRefCount
	if typeScriptImpactPrefersTypeRefs(def.Kind) {
		primaryRefCount = len(dedupeGenericRefs(refs.typeRefs))
	}

	switch {
	case !hasTests && nonTestRefCount >= typeScriptImpactHighNonTestReferenceThreshold:
		return goImpactRiskHigh
	case exported && !hasTests && primaryRefCount >= typeScriptImpactMediumReferenceThreshold:
		return goImpactRiskHigh
	case exported || !hasTests || nonTestRefCount >= typeScriptImpactMediumReferenceThreshold || primaryRefCount >= typeScriptImpactMediumReferenceThreshold:
		return goImpactRiskMedium
	default:
		return goImpactRiskLow
	}
}

func typeScriptDefinitionIsExported(def genericSymbolDef) bool {
	fields := strings.Fields(def.Signature)
	return len(fields) > 0 && fields[0] == "export"
}
