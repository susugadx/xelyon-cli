package review

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func resolveLexicalPath(baseDir, candidate string) string {
	if filepath.IsAbs(candidate) {
		return filepath.Clean(candidate)
	}
	return filepath.Clean(filepath.Join(baseDir, candidate))
}

func isPathWithinRepoRoot(repoRoot, resolvedPath string) (bool, error) {
	rel, err := filepath.Rel(repoRoot, resolvedPath)
	if err != nil {
		return false, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}

// resolvePathWithinRepoRoot は lexical 解決と repo root 内判定を行う。
func resolvePathWithinRepoRoot(repoRoot, baseDir, candidate string) (string, error) {
	resolved := resolveLexicalPath(baseDir, candidate)
	ok, err := isPathWithinRepoRoot(repoRoot, resolved)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %q: %w", candidate, err)
	}
	if !ok {
		return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked path %q is outside repository root", candidate))
	}
	return resolved, nil
}

// resolvePathWithinRepoRootWithSymlinkCheck は存在する path に対して symlink 解決後も repo root 内を保証する。
func resolvePathWithinRepoRootWithSymlinkCheck(repoRoot, baseDir, candidate string) (string, error) {
	resolved, err := resolvePathWithinRepoRoot(repoRoot, baseDir, candidate)
	if err != nil {
		return "", err
	}

	evaluated, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return resolved, nil
		}
		// 解決不能 path は lexical 判定で許可し、実行時にコマンド側の失敗へ委ねる。
		return resolved, nil
	}

	ok, err := isPathWithinRepoRoot(repoRoot, filepath.Clean(evaluated))
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink for %q: %w", candidate, err)
	}
	if !ok {
		return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked path %q resolves outside repository root", candidate))
	}

	return resolved, nil
}
