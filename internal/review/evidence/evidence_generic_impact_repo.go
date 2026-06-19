package evidence

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func (b *reviewGenericImpactCandidateBuilder) collectRepoPaths() {
	if strings.TrimSpace(b.repoRoot) == "" {
		return
	}
	if b.bundle.GenericImpactCandidatePathsCollected {
		b.collectRepoPathsFromCandidateList(b.bundle.GenericImpactCandidatePaths)
		return
	}
	err := filepath.WalkDir(b.repoRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			b.truncated = true
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		relPath, err := filepath.Rel(b.repoRoot, path)
		if err != nil {
			b.truncated = true
			return nil
		}
		relPath = filepath.ToSlash(relPath)
		if relPath == "." {
			return nil
		}
		if entry.IsDir() {
			if isReviewGenericImpactExcludedPath(relPath) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeType != 0 {
			return nil
		}
		if isReviewGenericImpactExcludedPath(relPath) {
			return nil
		}
		b.repoPaths = append(b.repoPaths, relPath)
		return nil
	})
	if err != nil {
		b.truncated = true
	}
	sort.Strings(b.repoPaths)
}

func (b *reviewGenericImpactCandidateBuilder) collectRepoPathsFromCandidateList(paths []string) {
	seen := make(map[string]struct{}, len(paths))
	for _, path := range paths {
		absPath, relPath, err := resolveReviewEvidenceRepoPathLexically(b.repoRoot, path)
		if err != nil || isReviewGenericImpactExcludedPath(relPath) {
			continue
		}
		if _, ok := seen[relPath]; ok {
			continue
		}
		info, err := os.Lstat(absPath)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		seen[relPath] = struct{}{}
		b.repoPaths = append(b.repoPaths, relPath)
	}
	sort.Strings(b.repoPaths)
}

func reviewGenericImpactChangedPathSet(bundle ReviewEvidenceBundle) map[string]struct{} {
	paths := make(map[string]struct{})
	add := func(path string) {
		relPath, ok := reviewGenericImpactBundleRelativePath(bundle.RepoRoot, path)
		if ok {
			paths[relPath] = struct{}{}
		}
	}
	for _, file := range bundle.ChangedFiles {
		add(file.Path)
		add(file.OldPath)
	}
	for _, file := range bundle.UntrackedFiles {
		add(file.Path)
	}
	for _, path := range bundle.Inventory.Untracked {
		add(path)
	}
	return paths
}

func reviewGenericImpactChangedDirs(paths map[string]struct{}) []string {
	dirs := make(map[string]struct{})
	for path := range paths {
		dirs[reviewGenericImpactPathDir(path)] = struct{}{}
	}
	return reviewGenericImpactSortedSet(dirs)
}

func reviewGenericImpactChangedStems(paths map[string]struct{}) []string {
	stems := make(map[string]struct{})
	for path := range paths {
		stem := reviewGenericImpactPathStem(path)
		if isReviewGenericImpactUsefulToken(stem) {
			stems[stem] = struct{}{}
		}
	}
	return reviewGenericImpactSortedSet(stems)
}

func reviewGenericImpactBundleRelativePath(repoRoot, path string) (string, bool) {
	if strings.TrimSpace(path) == "" {
		return "", false
	}
	if _, relPath, err := resolveReviewEvidenceRepoPathLexically(repoRoot, path); err == nil {
		return relPath, true
	}
	display := formatReviewEvidencePathDisplay(repoRoot, path)
	if display == "" || display == reviewEvidenceOutsideRepoPathDisplay || display == "." {
		return "", false
	}
	return display, true
}

func reviewGenericImpactAncestorDirSet(dirs []string) map[string]struct{} {
	ancestors := make(map[string]struct{})
	for _, dir := range dirs {
		for {
			if dir == "" {
				dir = "."
			}
			ancestors[dir] = struct{}{}
			if dir == "." {
				break
			}
			dir = reviewGenericImpactPathDir(dir)
		}
	}
	return ancestors
}
