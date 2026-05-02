package review

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type hostReadOnlyPathResolveOptions struct {
	repoRoot        string
	baseDir         string
	candidate       string
	evaluateSymlink bool
}

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

func resolvePathWithinRepoRootWithOptions(opts hostReadOnlyPathResolveOptions) (string, error) {
	resolved := resolveLexicalPath(opts.baseDir, opts.candidate)
	ok, err := isPathWithinRepoRoot(opts.repoRoot, resolved)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path %q: %w", opts.candidate, err)
	}
	if !ok {
		return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked path %q is outside repository root", opts.candidate))
	}

	if !opts.evaluateSymlink {
		return resolved, nil
	}

	evaluated, err := filepath.EvalSymlinks(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return resolved, nil
		}
		// 解決不能 path は lexical 判定で許可し、実行時にコマンド側の失敗へ委ねる。
		return resolved, nil
	}

	ok, err = isPathWithinRepoRoot(opts.repoRoot, filepath.Clean(evaluated))
	if err != nil {
		return "", fmt.Errorf("failed to resolve symlink for %q: %w", opts.candidate, err)
	}
	if !ok {
		return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked path %q resolves outside repository root", opts.candidate))
	}

	return resolved, nil
}

// resolvePathWithinRepoRoot は lexical 解決と repo root 内判定を行う。
func resolvePathWithinRepoRoot(repoRoot, baseDir, candidate string) (string, error) {
	return resolvePathWithinRepoRootWithOptions(hostReadOnlyPathResolveOptions{
		repoRoot:        repoRoot,
		baseDir:         baseDir,
		candidate:       candidate,
		evaluateSymlink: false,
	})
}

// resolvePathWithinRepoRootWithSymlinkCheck は存在する path に対して symlink 解決後も repo root 内を保証する。
func resolvePathWithinRepoRootWithSymlinkCheck(repoRoot, baseDir, candidate string) (string, error) {
	return resolvePathWithinRepoRootWithOptions(hostReadOnlyPathResolveOptions{
		repoRoot:        repoRoot,
		baseDir:         baseDir,
		candidate:       candidate,
		evaluateSymlink: true,
	})
}
