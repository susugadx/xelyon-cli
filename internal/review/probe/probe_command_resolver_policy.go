package probe

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
	seen := make(map[string]struct{}, len(roots)*2)
	for _, root := range roots {
		trimmed := strings.TrimSpace(root)
		if trimmed == "" {
			continue
		}

		absRoot, err := filepath.Abs(trimmed)
		if err != nil {
			return nil, newBlockedCommandErrorf("failed to resolve blocked root %q: %v", trimmed, err)
		}
		addNormalizedBlockedRoot(&normalized, seen, filepath.Clean(absRoot))

		evaluatedRoot, err := filepath.EvalSymlinks(absRoot)
		if err != nil {
			return nil, newBlockedCommandErrorf("failed to evaluate blocked root symlink %q: %v", absRoot, err)
		}
		evaluatedAbsRoot, err := filepath.Abs(evaluatedRoot)
		if err != nil {
			return nil, newBlockedCommandErrorf("failed to resolve blocked root symlink %q: %v", absRoot, err)
		}
		addNormalizedBlockedRoot(&normalized, seen, filepath.Clean(evaluatedAbsRoot))
	}
	return normalized, nil
}

func addNormalizedBlockedRoot(normalized *[]string, seen map[string]struct{}, root string) {
	key := resolverPathKey(root)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*normalized = append(*normalized, root)
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
	return resolverPathKey(a) == resolverPathKey(b)
}

func resolverPathKey(path string) string {
	cleaned := filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return strings.ToLower(cleaned)
	}
	return cleaned
}
