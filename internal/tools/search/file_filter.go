package search

import (
	"path"
	"path/filepath"
	"strings"
)

func ParseRawFileFilter(filter string) (string, string) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", ""
	}
	if containsGlobChar(filter) {
		return "", filter
	}
	return normalizeRawFileFilterToken(filter), ""
}

func MatchesRawFileFilter(filePath, filter string) bool {
	fileType, filePattern := ParseRawFileFilter(filter)
	return matchesFileFilterParts(filePath, fileType, filePattern)
}

func matchesFileFilterParts(filePath, fileType, filePattern string) bool {
	cleanPath := cleanFileFilterPath(filePath)
	globs := rawFileFilterGlobs(fileType, filePattern)
	if len(globs) == 0 {
		return true
	}
	return matchesAnyFileFilterGlob(cleanPath, globs)
}

func normalizeRawFileFilterToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, ".")
	return strings.ToLower(token)
}

func cleanFileFilterPath(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filePath))
}

func matchesAnyFileFilterGlob(cleanPath string, globs []string) bool {
	if cleanPath == "" {
		return false
	}
	baseName := path.Base(cleanPath)
	for _, glob := range globs {
		glob = filepath.ToSlash(strings.TrimSpace(glob))
		if glob == "" {
			continue
		}
		if matched, err := path.Match(glob, baseName); err == nil && matched {
			return true
		}
		if matched, err := path.Match(glob, cleanPath); err == nil && matched {
			return true
		}
	}
	return false
}
