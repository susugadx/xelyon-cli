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
