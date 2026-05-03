package review

import (
	"path/filepath"
	"runtime"
	"strings"
)

func rejectExecutableInsideBlockedRoots(resolvedPath string, roots ...string) error {
	normalizedRoots, err := normalizeBlockedRoots(roots)
	if err != nil {
		return err
	}

	if err := rejectExecutablePathInsideBlockedRoots(resolvedPath, normalizedRoots); err != nil {
		return err
	}

	evaluatedPath, err := filepath.EvalSymlinks(resolvedPath)
	if err != nil {
		return newBlockedCommandErrorf("failed to evaluate executable symlink %q: %v", resolvedPath, err)
	}
	evaluatedAbsPath, err := filepath.Abs(evaluatedPath)
	if err != nil {
		return newBlockedCommandErrorf("failed to resolve executable symlink %q: %v", resolvedPath, err)
	}
	evaluatedCleanPath := filepath.Clean(evaluatedAbsPath)
	if samePathForResolver(resolvedPath, evaluatedCleanPath) {
		return nil
	}

	return rejectExecutablePathInsideBlockedRoots(evaluatedCleanPath, normalizedRoots)
}

func normalizeBlockedRoots(roots []string) ([]string, error) {
	normalized := make([]string, 0, len(roots))
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}

		absRoot, err := filepath.Abs(trimmed)
		if err != nil {
			return nil, newBlockedCommandErrorf("failed to resolve blocked root %q: %v", trimmed, err)
		}
		normalized = append(normalized, filepath.Clean(absRoot))
	}
	return normalized, nil
}

func rejectExecutablePathInsideBlockedRoots(resolvedPath string, blockedRoots []string) error {
	for _, root := range blockedRoots {
		inside, err := isPathWithinRepoRoot(root, resolvedPath)
		if err != nil {
			return newBlockedCommandErrorf("failed to validate command path %q: %v", resolvedPath, err)
		}
		if inside {
			return newBlockedCommandErrorf("resolved executable %q is inside blocked root %q", resolvedPath, root)
		}
	}
	return nil
}

func samePathForResolver(a, b string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return filepath.Clean(a) == filepath.Clean(b)
}
