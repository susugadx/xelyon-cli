package search

import (
	"path/filepath"
	"strings"
)

func normalizeStructuredGoImpactScope(opts SearchOptions) (structuredImpactScope, bool) {
	definition, ok := normalizeStructuredGoImpactDefinitionOptions(opts)
	if !ok {
		return structuredImpactScope{}, false
	}
	return structuredImpactScope{
		Definition: definition,
		Evidence:   normalizeStructuredGoImpactEvidenceOptions(opts),
	}, true
}

func normalizeStructuredGoImpactDefinitionOptions(opts SearchOptions) (SearchOptions, bool) {
	fileType := strings.ToLower(strings.TrimSpace(opts.FileType))
	filePattern := cleanStructuredGoFilePattern(opts.FilePattern)

	if fileType != "" {
		if fileType != "go" {
			return SearchOptions{}, false
		}
		opts.FileType = "go"
		opts.FilePattern = ""
		return opts, true
	}

	if filePattern != "" {
		if !isStructuredGoImpactFilePattern(filePattern) {
			return SearchOptions{}, false
		}
		opts.FileType = ""
		opts.FilePattern = ""
		pathHint, ok := structuredGoImpactPathHintForFilePattern(opts, filePattern)
		if !ok {
			return SearchOptions{}, false
		}
		if pathHint != "" {
			opts.Path = pathHint
		}
		return opts, true
	}

	if isGoSourceFilePath(opts.Path) {
		return opts, true
	}
	if resolveLanguage(opts) == "go" {
		return opts, true
	}
	return SearchOptions{}, false
}

func normalizeStructuredGoImpactEvidenceOptions(opts SearchOptions) SearchOptions {
	fileType := strings.ToLower(strings.TrimSpace(opts.FileType))
	filePattern := cleanStructuredGoFilePattern(opts.FilePattern)

	if fileType != "" {
		opts.FileType = fileType
		opts.FilePattern = ""
		return opts
	}
	if filePattern != "" {
		opts.FileType = ""
		opts.FilePattern = filePattern
	}
	return opts
}

func cleanStructuredGoFilePattern(pattern string) string {
	return filepath.ToSlash(strings.TrimSpace(pattern))
}

func isGoSourceFilePath(filePath string) bool {
	clean := strings.ToLower(filepath.ToSlash(strings.TrimSpace(filePath)))
	return strings.HasSuffix(clean, ".go")
}

func isStructuredGoImpactFilePattern(pattern string) bool {
	pattern = strings.ToLower(cleanStructuredGoFilePattern(pattern))
	if pattern == "*.go" || pattern == "**/*.go" {
		return true
	}
	if !strings.HasSuffix(pattern, "/**/*.go") {
		return false
	}
	prefix := strings.TrimSuffix(pattern, "/**/*.go")
	return prefix != "" && !strings.ContainsAny(prefix, "*?[")
}
