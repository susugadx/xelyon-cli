package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func impactDefinitionSnippet(result navigation.InspectResult) string {
	if result.Symbol != nil && strings.TrimSpace(result.Symbol.Signature) != "" {
		return strings.TrimSpace(result.Symbol.Signature)
	}
	if len(result.Body) == 0 {
		return ""
	}
	line := strings.TrimSpace(result.Body[0])
	if idx := strings.Index(line, ":"); idx > 0 {
		allDigits := true
		for _, r := range line[:idx] {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if allDigits {
			return strings.TrimSpace(line[idx+1:])
		}
	}
	return line
}

func primaryCallerReadItems(refs []navigation.Reference, limit int) []SymbolBundleItem {
	if limit <= 0 || len(refs) == 0 {
		return nil
	}
	items := make([]SymbolBundleItem, 0, min(limit, len(refs)))
	for _, ref := range refs {
		if len(items) >= limit {
			break
		}
		snippet := strings.TrimSpace(ref.Snippet)
		if snippet == "" && ref.Scope != "" {
			snippet = ref.Scope
		}
		items = append(items, SymbolBundleItem{
			Kind:         "callers",
			File:         ref.File,
			ResolvedPath: ref.ResolvedPath,
			Line:         ref.Line,
			Snippet:      snippet,
			Scope:        ref.Scope,
		})
	}
	return items
}

func primaryTestReadItems(tests []navigation.TestRef, limit int) []SymbolBundleItem {
	if limit <= 0 || len(tests) == 0 {
		return nil
	}
	items := make([]SymbolBundleItem, 0, min(limit, len(tests)))
	for _, test := range tests {
		if len(items) >= limit {
			break
		}
		items = append(items, SymbolBundleItem{
			Kind:         "tests",
			File:         test.File,
			ResolvedPath: test.ResolvedPath,
			Line:         test.Line,
			Name:         test.Name,
			IsTest:       true,
		})
	}
	return items
}

func primaryImplementationReadItems(impls []navigation.ImplementationRef, limit int) []SymbolBundleItem {
	if limit <= 0 || len(impls) == 0 {
		return nil
	}
	items := make([]SymbolBundleItem, 0, min(limit, len(impls)))
	for _, impl := range impls {
		if len(items) >= limit {
			break
		}
		items = append(items, SymbolBundleItem{
			Kind:         "implementations",
			File:         impl.File,
			ResolvedPath: impl.ResolvedPath,
			Line:         impl.Line,
			Snippet:      strings.TrimSpace(impl.Name),
			Name:         impl.Name,
		})
	}
	return items
}

func crossPackageRefReadItems(result navigation.InspectResult, limit int) []SymbolBundleItem {
	if result.Symbol == nil || limit <= 0 || len(result.Refs) == 0 {
		return nil
	}

	symbolDir := filepath.ToSlash(strings.TrimSpace(result.Symbol.PackageDir))
	items := make([]SymbolBundleItem, 0, min(limit, len(result.Refs)))
	seenFiles := make(map[string]struct{})
	for _, ref := range result.Refs {
		refDir := filepath.ToSlash(filepath.Dir(strings.TrimSpace(ref.File)))
		if refDir == symbolDir {
			continue
		}
		if _, ok := seenFiles[ref.File]; ok {
			continue
		}
		seenFiles[ref.File] = struct{}{}
		snippet := strings.TrimSpace(ref.Snippet)
		if snippet == "" && ref.Scope != "" {
			snippet = ref.Scope
		}
		items = append(items, SymbolBundleItem{
			Kind:         "references",
			File:         ref.File,
			ResolvedPath: ref.ResolvedPath,
			Line:         ref.Line,
			Snippet:      snippet,
			Scope:        ref.Scope,
			IsTest:       ref.IsTest,
		})
		if len(items) >= limit {
			break
		}
	}
	return items
}
