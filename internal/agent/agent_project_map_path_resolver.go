package agent

import (
	"os"
	"path/filepath"
	"strings"
)

func projectMapPriorityPathsFromInput(cwd, rootPath string, candidates []string, limit int) []string {
	if limit <= 0 {
		return nil
	}

	capHint := len(candidates)
	if capHint > limit {
		capHint = limit
	}
	normalized := make([]string, 0, capHint)
	for _, candidate := range candidates {
		path, ok := resolveProjectMapInputCandidate(cwd, rootPath, candidate)
		if !ok {
			continue
		}
		normalized = append(normalized, path)
		if len(normalized) >= limit {
			break
		}
	}
	return normalized
}

func resolveProjectMapInputCandidate(cwd, rootPath, candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" || rootPath == "" {
		return "", false
	}
	if strings.HasPrefix(candidate, "http://") || strings.HasPrefix(candidate, "https://") {
		return "", false
	}
	if filepath.IsAbs(candidate) {
		absPath := filepath.Clean(candidate)
		if !projectMapPathExists(absPath) {
			return "", false
		}
		return canonicalizeProjectMapPriorityPath(rootPath, absPath)
	}
	if isWindowsAbsoluteProjectMapPath(candidate) {
		absPath := filepath.Clean(windowsAbsoluteProjectMapPathToLocal(candidate))
		if !projectMapPathExists(absPath) {
			return "", false
		}
		return canonicalizeProjectMapPriorityPath(rootPath, absPath)
	}

	sessionAbs := filepath.Clean(filepath.Join(cwd, filepath.FromSlash(candidate)))
	if isExplicitCWDRelativeProjectMapPath(candidate) {
		if strings.TrimSpace(cwd) == "" || !projectMapPathExists(sessionAbs) {
			return "", false
		}
		return canonicalizeProjectMapPriorityPath(rootPath, sessionAbs)
	}
	rootAbs := filepath.Clean(filepath.Join(rootPath, filepath.FromSlash(candidate)))

	sessionExists := projectMapPathExists(sessionAbs)
	rootExists := projectMapPathExists(rootAbs)

	switch {
	case rootExists && (looksRepoRelativeProjectMapPath(candidate) || !sessionExists):
		return canonicalizeProjectMapPriorityPath(rootPath, rootAbs)
	case sessionExists:
		return canonicalizeProjectMapPriorityPath(rootPath, sessionAbs)
	case rootExists:
		return canonicalizeProjectMapPriorityPath(rootPath, rootAbs)
	default:
		return "", false
	}
}

func canonicalizeProjectMapPriorityPath(rootPath, absPath string) (string, bool) {
	if rootPath == "" || absPath == "" {
		return "", false
	}

	rootAbs, err := filepath.Abs(rootPath)
	if err != nil {
		return "", false
	}
	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", false
	}

	relPath, err := filepath.Rel(rootAbs, absPath)
	if err != nil {
		return "", false
	}
	if relPath == "." {
		return "", false
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}

	return filepath.ToSlash(filepath.Clean(relPath)), true
}

func projectMapPathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func looksRepoRelativeProjectMapPath(candidate string) bool {
	candidate = filepath.ToSlash(strings.TrimSpace(candidate))
	if candidate == "" {
		return false
	}
	if strings.HasPrefix(candidate, "./") || strings.HasPrefix(candidate, "../") {
		return false
	}
	return strings.Contains(candidate, "/")
}

func isExplicitCWDRelativeProjectMapPath(candidate string) bool {
	candidate = strings.ReplaceAll(filepath.ToSlash(strings.TrimSpace(candidate)), "\\", "/")
	return strings.HasPrefix(candidate, "./")
}

func isWindowsAbsoluteProjectMapPath(candidate string) bool {
	if len(candidate) < 4 {
		return false
	}
	if (candidate[0] < 'A' || candidate[0] > 'Z') && (candidate[0] < 'a' || candidate[0] > 'z') {
		return false
	}
	return candidate[1] == ':' && candidate[2] == '/'
}

func windowsAbsoluteProjectMapPathToLocal(candidate string) string {
	if !isWindowsAbsoluteProjectMapPath(candidate) {
		return candidate
	}
	return candidate[2:]
}
