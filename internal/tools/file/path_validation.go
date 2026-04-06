package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func resolveValidatedPath(out common.Output, path, emptyMessage string) (string, string) {
	return resolveValidatedPathWithRoots(out, path, nil, emptyMessage)
}

func resolveValidatedPathWithRoots(out common.Output, path string, allowedRoots []string, emptyMessage string) (string, string) {
	if path == "" {
		return "", "Error: " + emptyMessage
	}

	var (
		absPath string
		err     error
	)
	if len(allowedRoots) > 0 {
		absPath, err = validatePathWithinRoots(path, allowedRoots)
	} else {
		absPath, err = common.ValidatePath(path)
	}
	if err != nil {
		out.Red.Printf("🚫 Security: %v\n", err)
		return "", fmt.Sprintf("Error: %v", err)
	}
	return absPath, ""
}

func validatePathWithinRoots(path string, allowedRoots []string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path is empty")
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	cleanPath := filepath.Clean(absPath)

	normalizedRoots := normalizeAllowedRoots(allowedRoots)
	if len(normalizedRoots) == 0 {
		return common.ValidatePath(path)
	}
	realRoots := evalAllowedRoots(normalizedRoots)
	comparisonRoots := mergeAllowedRoots(normalizedRoots, realRoots)
	if !pathWithinAnyRoot(cleanPath, comparisonRoots) {
		return "", fmt.Errorf("path escape attempt detected: %s is outside of %s", cleanPath, strings.Join(comparisonRoots, ", "))
	}

	if _, statErr := os.Lstat(cleanPath); statErr == nil {
		realPath, err := filepath.EvalSymlinks(cleanPath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve symlinks: %w", err)
		}
		if !pathWithinAnyRoot(realPath, realRoots) {
			return "", fmt.Errorf("symlink escape attempt detected: %s resolves to %s (outside %s)", cleanPath, realPath, strings.Join(realRoots, ", "))
		}
		return realPath, nil
	}

	parentDir := filepath.Dir(cleanPath)
	if _, statErr := os.Lstat(parentDir); statErr == nil {
		realParent, err := filepath.EvalSymlinks(parentDir)
		if err == nil {
			realRoots := evalAllowedRoots(normalizedRoots)
			if !pathWithinAnyRoot(realParent, realRoots) {
				return "", fmt.Errorf("symlink escape attempt in parent directory: %s", parentDir)
			}
		}
	}

	return cleanPath, nil
}

func normalizeAllowedRoots(roots []string) []string {
	normalized := make([]string, 0, len(roots))
	for _, root := range roots {
		root = normalizeWorkspaceRoot(root)
		if root == "" {
			continue
		}
		normalized = appendUniqueString(normalized, root)
	}
	return normalized
}

func mergeAllowedRoots(groups ...[]string) []string {
	merged := make([]string, 0)
	for _, group := range groups {
		for _, root := range group {
			if root == "" || containsString(merged, root) {
				continue
			}
			merged = append(merged, root)
		}
	}
	return merged
}

func evalAllowedRoots(roots []string) []string {
	evaluated := make([]string, 0, len(roots))
	for _, root := range roots {
		evaluated = appendUniqueString(evaluated, evaluateWorkspaceRoot(root))
	}
	return evaluated
}

func pathWithinAnyRoot(target string, roots []string) bool {
	for _, root := range roots {
		if isPathWithinRoot(target, root) {
			return true
		}
	}
	return false
}

func isPathWithinRoot(target, root string) bool {
	if target == root {
		return true
	}
	return strings.HasPrefix(target, root+string(filepath.Separator))
}
