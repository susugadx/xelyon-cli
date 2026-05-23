package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/jsast"
)

const (
	jsFamilyImpactRecommendedReadLimit            = 5
	jsFamilyImpactRecommendedReadPerGroupLimit    = 1
	jsFamilyImpactHighNonTestReferenceThreshold   = 8
	jsFamilyImpactMediumNonTestReferenceThreshold = 4
)

type jsFamilyImpactReadGroup struct {
	kind   string
	refs   []genericSymbolRef
	limit  int
	isTest bool
}

type jsFamilyImpactBundleSpec struct {
	language    string
	debugSource string
	symbol      string
	def         genericSymbolDef
	rootPath    string
	impact      *SymbolBundleImpact
}

func newJSFamilyImpactBundle(spec jsFamilyImpactBundleSpec) *SymbolBundle {
	if spec.impact == nil {
		return nil
	}
	displayName := spec.def.Name
	if displayName == "" {
		displayName = spec.symbol
	}
	return &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    spec.language,
			Query:       spec.symbol,
			Canonical:   canonicalSymbolBundleKey(spec.language, spec.def.File, spec.def.Line, displayName),
			DisplayName: displayName,
			Kind:        spec.def.Kind,
			File:        spec.def.File,
			Line:        spec.def.Line,
			EndLine:     spec.def.Line,
		},
		Definition: SymbolBundleDefinition{
			File:      spec.def.File,
			Line:      spec.def.Line,
			EndLine:   spec.def.Line,
			Signature: spec.def.Signature,
			Body:      []string{fmt.Sprintf("%d: %s", spec.def.Line, spec.def.Signature)},
		},
		Impact: spec.impact,
		Debug: SymbolBundleDebug{
			Source:       spec.debugSource,
			FileRootPath: spec.rootPath,
		},
	}
}

func buildJSFamilyImpactMetadata(def genericSymbolDef, rootPath string, riskLevel string, groups []jsFamilyImpactReadGroup) *SymbolBundleImpact {
	impact := &SymbolBundleImpact{
		RiskLevel:        riskLevel,
		RecommendedReads: make([]SymbolBundleItem, 0, jsFamilyImpactRecommendedReadLimit),
	}

	seen := make(map[string]struct{}, jsFamilyImpactRecommendedReadLimit)
	add := func(item SymbolBundleItem) {
		if item.File == "" || item.Line <= 0 || len(impact.RecommendedReads) >= jsFamilyImpactRecommendedReadLimit {
			return
		}
		key := jsFamilyImpactLocationKey(item.File, item.Line)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		impact.RecommendedReads = append(impact.RecommendedReads, item)
	}

	add(jsFamilyImpactDefinitionItem(def, rootPath))
	for _, group := range groups {
		for _, item := range jsFamilyImpactItemsFromRefs(def, group.refs, group.kind, group.limit, group.isTest, rootPath, def.Name) {
			add(item)
		}
	}

	if len(impact.RecommendedReads) == 0 {
		return nil
	}
	return impact
}

func appendJSFamilyImpactSectionWithTotals(bundle *SymbolBundle, def genericSymbolDef, kind, title string, refs []genericSymbolRef, totalRefs []genericSymbolRef, limit int, isTest bool, rootPath string, symbol string) {
	items := jsFamilyImpactItemsFromRefs(def, refs, kind, limit, isTest, rootPath, symbol)
	if len(items) == 0 {
		return
	}
	if totalRefs == nil {
		totalRefs = refs
	}

	total := len(dedupeGenericRefs(totalRefs))
	bundle.Sections = append(bundle.Sections, SymbolBundleSection{
		Kind:  kind,
		Title: title,
		Items: items,
		Total: total,
		More:  total > len(items),
	})
}

func jsFamilyImpactItemsFromRefs(def genericSymbolDef, refs []genericSymbolRef, kind string, limit int, isTest bool, rootPath string, symbol string) []SymbolBundleItem {
	selected := prioritizeGenericRefs(def, refs, limit, isTest)
	if len(selected) == 0 {
		return nil
	}

	items := make([]SymbolBundleItem, 0, len(selected))
	for _, ref := range selected {
		items = append(items, jsFamilyImpactItemFromRef(kind, ref, rootPath, symbol, isTest))
	}
	return items
}

func jsFamilyImpactDefinitionItem(def genericSymbolDef, rootPath string) SymbolBundleItem {
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

func jsFamilyImpactItemFromRef(kind string, ref genericSymbolRef, rootPath string, symbol string, forceTest bool) SymbolBundleItem {
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

func jsFamilyImpactLocationKey(file string, line int) string {
	if file == "" || line <= 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func jsFamilySignatureStartsWithExport(signature string) bool {
	fields := strings.Fields(signature)
	return len(fields) > 0 && fields[0] == "export"
}

func jsFamilyRefsContainExport(refs []genericSymbolRef) bool {
	for _, ref := range refs {
		if ref.Class == jsast.ClassExport {
			return true
		}
	}
	return false
}
