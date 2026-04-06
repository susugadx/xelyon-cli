package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

type primaryFileRef struct {
	DisplayPath  string
	ResolvedPath string
}

type primaryFileRefSource int

const (
	primaryFileRefSourceText primaryFileRefSource = iota
	primaryFileRefSourceStructuredSymbol
	primaryFileRefSourceInvocationRelative
)

func newSearchLocator(displayPath, resolvedPath string, line, endLine int, name string) locator.Location {
	return locator.Location{
		FilePath:     displayPath,
		ResolvedPath: cleanResolvedLocatorPath(resolvedPath),
		Line:         line,
		EndLine:      endLine,
		Name:         name,
	}
}

func newTextSearchLocator(displayPath string, line, endLine int, name string, opts SearchOptions) locator.Location {
	return newSearchLocator(displayPath, absoluteAffectedFilePath(displayPath, opts, affectedFileSourceText), line, endLine, name)
}

func newBundleLocator(displayPath string, line, endLine int, name string, bundle *SymbolBundle) locator.Location {
	return newBundleScopedLocator(displayPath, "", line, endLine, name, bundle)
}

func newBundleItemLocator(item SymbolBundleItem, bundle *SymbolBundle) locator.Location {
	return newBundleScopedLocator(item.File, item.ResolvedPath, item.Line, item.EndLine, item.Name, bundle)
}

func newBundleScopedLocator(displayPath, resolvedPath string, line, endLine int, name string, bundle *SymbolBundle) locator.Location {
	if clean := cleanResolvedLocatorPath(resolvedPath); clean != "" {
		return newSearchLocator(displayPath, clean, line, endLine, name)
	}
	rootPath := ""
	if bundle != nil {
		rootPath = bundle.Debug.FileRootPath
	}
	return newSearchLocator(displayPath, absoluteAffectedFilePathWithBase(displayPath, rootPath), line, endLine, name)
}

func extractPrimaryFilePaths(output string) []string {
	refs := extractPrimaryFileRefs(output, SearchOptions{})
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.DisplayPath)
	}
	return paths
}

func extractPrimaryFileRefs(output string, opts SearchOptions) []primaryFileRef {
	var refs []primaryFileRef
	seen := make(map[string]bool)
	add := func(file string, source primaryFileRefSource) {
		displayPath := strings.TrimSpace(file)
		if displayPath == "" {
			return
		}

		resolvedPath := resolvePrimaryFileRefPath(displayPath, opts, source)

		key := displayPath + "\x00" + cleanResolvedLocatorPath(resolvedPath)
		if seen[key] {
			return
		}
		seen[key] = true
		refs = append(refs, primaryFileRef{
			DisplayPath:  displayPath,
			ResolvedPath: cleanResolvedLocatorPath(resolvedPath),
		})
	}

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "📄 ") {
			rest := strings.TrimPrefix(trimmed, "📄 ")
			if idx := strings.Index(rest, " ("); idx > 0 {
				add(rest[:idx], primaryFileRefSourceText)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "── ") && strings.Contains(trimmed, " in ") && strings.HasSuffix(trimmed, "──") {
			inIdx := strings.LastIndex(trimmed, " in ")
			rest := trimmed[inIdx+4:]
			rest = strings.TrimSuffix(rest, "──")
			rest = trimRenderedPrimaryFilePath(rest)
			add(rest, primaryFileRefSourceStructuredSymbol)
			continue
		}
		if hasNumericListPrefix(trimmed) {
			if numbered, ok := parseNumberedCandidateFilePath(trimmed); ok {
				add(trimRenderedPrimaryFilePath(numbered), primaryFileRefSourceStructuredSymbol)
				continue
			}
			if idx := strings.LastIndex(trimmed, " in "); idx > 0 {
				add(trimRenderedPrimaryFilePath(trimmed[idx+4:]), primaryFileRefSourceInvocationRelative)
				continue
			}
		}
	}
	return refs
}

func resolvePrimaryFileRefPath(displayPath string, opts SearchOptions, source primaryFileRefSource) string {
	switch source {
	case primaryFileRefSourceStructuredSymbol:
		return absoluteAffectedFilePathForSymbol(displayPath, opts, "")
	case primaryFileRefSourceInvocationRelative:
		return absoluteAffectedFilePathWithBase(displayPath, invocationCWDOrGetwd(opts))
	default:
		return absoluteAffectedFilePath(displayPath, opts, affectedFileSourceText)
	}
}

func cleanResolvedLocatorPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func trimRenderedPrimaryFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if atIdx := strings.LastIndex(path, " @"); atIdx > 0 {
		path = strings.TrimSpace(path[:atIdx])
	}
	for {
		lIdx := strings.LastIndex(path, " [L")
		if lIdx <= 0 || !strings.HasSuffix(path, "]") {
			break
		}
		path = strings.TrimSpace(path[:lIdx])
	}
	return path
}
