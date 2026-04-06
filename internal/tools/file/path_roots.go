package file

import (
	"path/filepath"
	"strings"
)

func normalizeWorkspaceRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func evaluateWorkspaceRoot(path string) string {
	path = normalizeWorkspaceRoot(path)
	if path == "" {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return filepath.Clean(resolved)
	}
	return path
}

func appendUniqueString(values []string, candidate string) []string {
	if candidate == "" || containsString(values, candidate) {
		return values
	}
	return append(values, candidate)
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
