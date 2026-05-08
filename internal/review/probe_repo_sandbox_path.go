package review

import (
	"os"
	"path/filepath"
	"strings"
)

func validateRepoSandboxExistingAncestorsWithinWorktree(worktreeDir, absPath, label string) error {
	return validateModeExistingAncestorsWithinRoot(worktreeDir, absPath, label, "repo_sandbox", "sandbox worktree")
}

func validateScratchExistingAncestorsWithinRoot(scratchDir, absPath, label string) error {
	return validateModeExistingAncestorsWithinRoot(scratchDir, absPath, label, "scratch", "scratch directory")
}

func validateModeExistingAncestorsWithinRoot(rootDir, absPath, label, modeName, rootLabel string) error {
	rootDir = filepath.Clean(rootDir)
	parent := filepath.Clean(filepath.Dir(absPath))

	rel, err := filepath.Rel(rootDir, parent)
	if err != nil {
		return newBlockedCommandErrorf("failed to validate %s %s parent: %v", modeName, label, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return newBlockedCommandErrorf("%s %s parent escapes %s", modeName, label, rootLabel)
	}
	if rel == "." {
		return nil
	}

	current := rootDir
	for _, part := range strings.Split(rel, string(filepath.Separator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)

		if err := validateModeExistingAncestorWithinRoot(rootDir, current, label, modeName, rootLabel); err != nil {
			return err
		}
	}
	return nil
}

func validateModeExistingAncestorWithinRoot(rootDir, current, label, modeName, rootLabel string) error {
	info, err := os.Lstat(current)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return newBlockedCommandErrorf("failed to inspect %s %s parent %q: %v", modeName, label, current, err)
	}

	evaluated, err := filepath.EvalSymlinks(current)
	if err != nil {
		return newBlockedCommandErrorf("failed to evaluate %s %s parent %q: %v", modeName, label, current, err)
	}
	inside, err := isPathWithinRepoRoot(rootDir, filepath.Clean(evaluated))
	if err != nil {
		return newBlockedCommandErrorf("failed to validate %s %s parent %q: %v", modeName, label, current, err)
	}
	if !inside {
		return newBlockedCommandErrorf("%s %s parent escapes %s", modeName, label, rootLabel)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		targetInfo, err := os.Stat(current)
		if err != nil {
			return newBlockedCommandErrorf("failed to stat %s %s parent %q: %v", modeName, label, current, err)
		}
		if !targetInfo.IsDir() {
			return newBlockedCommandErrorf("%s %s parent %q is not a directory", modeName, label, current)
		}
		return nil
	}
	if !info.IsDir() {
		return newBlockedCommandErrorf("%s %s parent %q is not a directory", modeName, label, current)
	}
	return nil
}
