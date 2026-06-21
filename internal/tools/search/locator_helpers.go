package search

import "github.com/susugadx/xelyon-cli/internal/locator"

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
