package search

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

func buildGenericBundleSection(def genericSymbolDef, input symbolBundleSectionInput) *SymbolBundleSection {
	if len(input.Items) == 0 {
		return nil
	}

	totalItems := input.Items
	if input.TotalItems != nil {
		totalItems = input.TotalItems
	}
	total := len(dedupeGenericRefs(totalItems))
	items := prioritizeGenericRefs(def, input.Items, input.Limit, input.IsTest)
	if len(items) == 0 {
		return nil
	}

	sectionItems := make([]SymbolBundleItem, 0, len(items))
	for _, item := range items {
		sectionItems = append(sectionItems, SymbolBundleItem{
			Kind:    input.Kind,
			File:    item.File,
			Line:    item.Line,
			Snippet: strings.TrimSpace(item.Snippet),
			IsTest:  input.IsTest || item.IsTest,
		})
	}

	return &SymbolBundleSection{
		Kind:  input.Kind,
		Title: input.Title,
		Items: sectionItems,
		Total: total,
		More:  total > len(sectionItems),
	}
}

func dedupeGenericRefs(refs []genericSymbolRef) []genericSymbolRef {
	seen := make(map[string]bool)
	result := make([]genericSymbolRef, 0, len(refs))
	for _, ref := range refs {
		key := fmt.Sprintf("%s:%d", ref.File, ref.Line)
		if seen[key] {
			continue
		}
		seen[key] = true
		result = append(result, ref)
	}
	return result
}

func prioritizeGenericRefs(def genericSymbolDef, refs []genericSymbolRef, limit int, testOnly bool) []genericSymbolRef {
	if limit <= 0 {
		return nil
	}

	uniq := dedupeGenericRefs(refs)
	sort.SliceStable(uniq, func(i, j int) bool {
		left := genericRefScore(def, uniq[i], testOnly)
		right := genericRefScore(def, uniq[j], testOnly)
		if left != right {
			return left > right
		}
		if uniq[i].File != uniq[j].File {
			return uniq[i].File < uniq[j].File
		}
		return uniq[i].Line < uniq[j].Line
	})

	selected := make([]genericSymbolRef, 0, min(limit, len(uniq)))
	seenFile := make(map[string]bool)
	for _, ref := range uniq {
		if len(selected) >= limit {
			break
		}
		if seenFile[ref.File] {
			continue
		}
		seenFile[ref.File] = true
		selected = append(selected, ref)
	}
	for _, ref := range uniq {
		if len(selected) >= limit {
			break
		}
		if containsGenericRef(selected, ref) {
			continue
		}
		selected = append(selected, ref)
	}
	return selected
}

func containsGenericRef(refs []genericSymbolRef, target genericSymbolRef) bool {
	for _, ref := range refs {
		if ref.File == target.File && ref.Line == target.Line {
			return true
		}
	}
	return false
}

func genericRefScore(def genericSymbolDef, ref genericSymbolRef, testOnly bool) int {
	score := 0
	defDir := filepath.Dir(def.File)
	refDir := filepath.Dir(ref.File)
	if defDir == refDir {
		score += 40
	}
	if ref.File == def.File {
		score += 30
	}
	if strings.Contains(strings.ToLower(filepath.Base(ref.File)), strings.ToLower(def.Name)) {
		score += 20
	}
	if strings.Contains(strings.ToLower(filepath.Base(ref.File)), strings.ToLower(strings.TrimSuffix(filepath.Base(def.File), filepath.Ext(def.File)))) {
		score += 15
	}
	if testOnly {
		if ref.IsTest {
			score += 50
		}
	} else if ref.IsTest {
		score -= 100
	}
	if classifyFilePath(ref.File) == "impl" {
		score += 5
	}
	score -= min(ref.Line, 200)
	return score
}
