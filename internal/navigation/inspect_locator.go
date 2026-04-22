package navigation

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func newInspectSymbolLocator(filePath, rootPath string, line, endLine int, name string) locator.Location {
	return locator.Location{
		FilePath:     filePath,
		ResolvedPath: resolveInspectLocatorPath(filePath, rootPath),
		Line:         line,
		EndLine:      endLine,
		Name:         name,
	}
}

func newInspectRelatedLocator(filePath, resolvedPath, rootPath string, line, endLine int, name string) locator.Location {
	if strings.TrimSpace(resolvedPath) == "" {
		resolvedPath = resolveInspectLocatorPath(filePath, rootPath)
	} else {
		resolvedPath = cleanInspectResolvedPath(resolvedPath)
	}
	return locator.Location{
		FilePath:     filePath,
		ResolvedPath: resolvedPath,
		Line:         line,
		EndLine:      endLine,
		Name:         name,
	}
}

func resolveInspectLocatorPath(filePath, rootPath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	if filepath.IsAbs(filePath) {
		return filepath.Clean(filePath)
	}
	filePath = cleanRelativeNavigationPath(filePath)
	rootPath = normalizeInspectRootPath(rootPath)
	if rootPath == "" {
		return ""
	}
	resolved, ok := resolveRelativePathAgainstBase(rootPath, filePath)
	if !ok {
		return ""
	}
	return filepath.Clean(resolved)
}
