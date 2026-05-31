package probe

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/review/pathpolicy"
)

func resolvePathWithinRepoRootLexically(repoRoot, baseDir, candidate string) (resolved string, outsideRepo bool, err error) {
	return pathpolicy.ResolveWithinRootLexically(repoRoot, baseDir, candidate)
}

func isPathWithinRepoRoot(repoRoot, resolvedPath string) (bool, error) {
	return pathpolicy.IsWithinRoot(repoRoot, resolvedPath)
}

func checkSymlinkResolutionWithinRepoRoot(repoRoot, resolvedPath string) (outsideRepo bool, err error) {
	return pathpolicy.CheckSymlinkResolutionWithinRoot(repoRoot, resolvedPath)
}

// resolvePathWithinRepoRoot は lexical 解決と repo root 内判定を行う。
func resolvePathWithinRepoRoot(repoRoot, baseDir, candidate string) (string, error) {
	resolved, outside, err := resolvePathWithinRepoRootLexically(repoRoot, baseDir, candidate)
	if err != nil {
		return "", err
	}
	if outside {
		return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked path %q is outside repository root", candidate))
	}
	return resolved, nil
}

// resolvePathWithinRepoRootWithSymlinkCheck は存在する path に対して symlink 解決後も repo root 内を保証する。
func resolvePathWithinRepoRootWithSymlinkCheck(repoRoot, baseDir, candidate string) (string, error) {
	resolved, outside, err := resolvePathWithinRepoRootLexically(repoRoot, baseDir, candidate)
	if err != nil {
		return "", err
	}
	if outside {
		return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked path %q is outside repository root", candidate))
	}

	outside, err = checkSymlinkResolutionWithinRepoRoot(repoRoot, resolved)
	if err != nil {
		return "", err
	}
	if outside {
		return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked path %q resolves outside repository root", candidate))
	}

	return resolved, nil
}
