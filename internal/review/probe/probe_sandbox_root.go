package probe

import "path/filepath"

func validateIsolatedRootOutsideRepo(repoRoot, rootPath, modeName string) error {
	resolvedRoot := filepath.Clean(rootPath)
	if !filepath.IsAbs(resolvedRoot) {
		abs, err := filepath.Abs(resolvedRoot)
		if err != nil {
			return newBlockedCommandErrorf("failed to resolve %s root %q: %v", modeName, rootPath, err)
		}
		resolvedRoot = filepath.Clean(abs)
	}

	insideRepo, err := isPathWithinRepoRoot(repoRoot, resolvedRoot)
	if err != nil {
		return newBlockedCommandErrorf("failed to validate %s root %q: %v", modeName, rootPath, err)
	}
	if insideRepo {
		return newBlockedCommandErrorf("%s root must be outside repository root: %s", modeName, resolvedRoot)
	}
	return nil
}
