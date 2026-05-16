package search

import (
	"os"
	"path/filepath"
	"strings"
)

func jsFamilySearchCandidateAllowed(absPath string, displayPath string, opts SearchOptions) bool {
	return searchCandidateAllowedByOptions(absPath, displayPath, opts)
}

func searchCandidateAllowedByOptions(absPath string, displayPath string, opts SearchOptions) bool {
	if !searchPathInScope(absPath, opts) {
		return false
	}
	if matchesSearchIgnoreFilter(displayPath, opts) {
		return false
	}
	return matchesSearchFileFilter(displayPath, opts)
}

func searchPathInScope(absPath string, opts SearchOptions) bool {
	basis := resolveSearchPathBasisForOptions(opts)
	base := basis.Workdir
	if strings.TrimSpace(base) == "" {
		base = invocationCWDOrGetwd(opts)
	}
	target := strings.TrimSpace(basis.Target)
	if target == "" {
		target = "."
	}

	var targetPath string
	if filepath.IsAbs(target) {
		targetPath = filepath.Clean(target)
	} else {
		targetPath = filepath.Clean(filepath.Join(base, target))
	}

	info, err := os.Stat(targetPath)
	if err == nil && !info.IsDir() {
		return filepath.Clean(absPath) == targetPath
	}

	rel, err := filepath.Rel(targetPath, filepath.Clean(absPath))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
