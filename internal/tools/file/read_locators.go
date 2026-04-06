package file

import "github.com/susugadx/xelyon-cli/internal/locator"

func newReadResultLocator(filePath, resolvedPath string, startLine, endLine int, name string) locator.Location {
	return locator.Location{
		FilePath:     filePath,
		ResolvedPath: resolvedPath,
		Line:         startLine,
		EndLine:      endLine,
		Name:         name,
	}
}

func newReadResultLocatorForBatch(result readFileBatchResult) locator.Location {
	return newReadResultLocator(result.filePath, result.resolvedPath, result.startLine, result.endLine, result.locatorName)
}
