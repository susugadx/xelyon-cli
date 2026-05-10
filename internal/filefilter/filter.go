package filefilter

import (
	"path"
	"path/filepath"
	"strings"
)

func Parse(filter string) (string, string) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", ""
	}
	if containsGlobChar(filter) {
		return "", filter
	}
	return normalizeRawFileFilterToken(filter), ""
}

func containsGlobChar(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

func Matches(filePath, filter string) bool {
	fileType, filePattern := Parse(filter)
	return MatchesParts(filePath, fileType, filePattern)
}

// MatchesParts は parse 済みの file type / glob pattern で path を判定する。
func MatchesParts(filePath, fileType, filePattern string) bool {
	cleanPath := cleanFileFilterPath(filePath)
	globs := Globs(fileType, filePattern)
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

// CleanPath は file_filter matching 用に path 表現を slash 区切りへ正規化する。
func CleanPath(filePath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filePath))
}

func cleanFileFilterPath(filePath string) string {
	return CleanPath(filePath)
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
		if strings.Contains(glob, "**") && matchDoubleStarGlob(glob, cleanPath) {
			return true
		}
	}
	return false
}

func matchDoubleStarGlob(glob, cleanPath string) bool {
	glob = strings.Trim(glob, "/")
	cleanPath = strings.Trim(cleanPath, "/")
	if glob == "" || cleanPath == "" {
		return false
	}
	return matchGlobSegments(strings.Split(glob, "/"), strings.Split(cleanPath, "/"))
}

func matchGlobSegments(patterns, parts []string) bool {
	if len(patterns) == 0 {
		return len(parts) == 0
	}

	if patterns[0] == "**" {
		for i := 0; i <= len(parts); i++ {
			if matchGlobSegments(patterns[1:], parts[i:]) {
				return true
			}
		}
		return false
	}

	if len(parts) == 0 {
		return false
	}
	matched, err := path.Match(patterns[0], parts[0])
	if err != nil || !matched {
		return false
	}
	return matchGlobSegments(patterns[1:], parts[1:])
}
