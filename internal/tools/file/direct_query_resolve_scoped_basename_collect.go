package file

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func resolveScopedBasenameDirectTarget(scopes []scopedExactLookupScope, ignoreMatcher *pathmatch.Matcher, entry directQueryEntryInput, fileFilter string) scopedDirectTargetOutcome {
	trimmedFilter := strings.TrimSpace(fileFilter)
	matches := make([]DirectQueryTarget, 0, 1)
	seen := make(map[string]struct{})
	for _, scope := range scopes {
		limit := 0
		if trimmedFilter == "" {
			limit = 2 - len(matches)
		}
		for _, target := range collectScopedBasenameTargets(scope, ignoreMatcher, entry, fileFilter, seen, limit) {
			matches = append(matches, target)
			if trimmedFilter == "" && len(matches) > 1 {
				return scopedDirectTargetOutcome{Kind: scopedDirectResolutionAmbiguous}
			}
		}
	}
	return selectScopedBasenameDirectTarget(matches, trimmedFilter, entry.rawEntry)
}

func collectScopedBasenameTargets(scope scopedExactLookupScope, ignoreMatcher *pathmatch.Matcher, entry directQueryEntryInput, fileFilter string, seen map[string]struct{}, limit int) []DirectQueryTarget {
	if limit <= 0 {
		limit = 0
	}

	info, err := os.Stat(scope.resolvedPath)
	if err != nil {
		return nil
	}

	if !info.IsDir() {
		target, ok := buildScopedBasenameTarget(scope, scope.resolvedPath, entry, fileFilter, seen)
		if !ok {
			return nil
		}
		return []DirectQueryTarget{target}
	}

	matches := make([]DirectQueryTarget, 0, 1)
	_ = filepath.WalkDir(scope.resolvedPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if path != scope.resolvedPath {
			relPath, ok := relativeScopedLookupDisplayPath(scope.displayRoot, path)
			if ok && ignoreMatcher != nil && ignoreMatcher.Match(relPath, d.IsDir()) {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
		}
		if d.IsDir() {
			return nil
		}

		target, ok := buildScopedBasenameTarget(scope, path, entry, fileFilter, seen)
		if !ok {
			return nil
		}

		matches = append(matches, target)
		if limit > 0 && len(matches) >= limit {
			return fs.SkipAll
		}
		return nil
	})
	return matches
}

func buildScopedBasenameTarget(scope scopedExactLookupScope, candidatePath string, entry directQueryEntryInput, fileFilter string, seen map[string]struct{}) (DirectQueryTarget, bool) {
	if filepath.Base(candidatePath) != entry.cleanedPath {
		return DirectQueryTarget{}, false
	}

	resolvedPath, ok := resolveExistingScopedLookupPath(candidatePath, []string{scope.resolvedPath})
	if !ok {
		return DirectQueryTarget{}, false
	}
	if _, exists := seen[resolvedPath]; exists {
		return DirectQueryTarget{}, false
	}

	target, ok := buildScopedTargetFromResolvedPath(scope, resolvedPath, entry, fileFilter)
	if !ok {
		return DirectQueryTarget{}, false
	}
	seen[resolvedPath] = struct{}{}
	return target, true
}
