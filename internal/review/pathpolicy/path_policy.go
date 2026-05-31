package pathpolicy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveLexical は baseDir から candidate を lexical に解決する。
func ResolveLexical(baseDir, candidate string) string {
	if filepath.IsAbs(candidate) {
		return filepath.Clean(candidate)
	}
	return filepath.Clean(filepath.Join(baseDir, candidate))
}

// IsWithinRoot は resolvedPath が root 配下にあるかを判定する。
func IsWithinRoot(root, resolvedPath string) (bool, error) {
	rel, err := filepath.Rel(root, resolvedPath)
	if err != nil {
		return false, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false, nil
	}
	return true, nil
}

// ResolveWithinRootLexically は candidate を lexical 解決し、root 外かどうかを返す。
func ResolveWithinRootLexically(root, baseDir, candidate string) (resolved string, outsideRoot bool, err error) {
	resolved = ResolveLexical(baseDir, candidate)
	ok, err := IsWithinRoot(root, resolved)
	if err != nil {
		return "", false, fmt.Errorf("failed to resolve path %q: %w", candidate, err)
	}
	if !ok {
		return resolved, true, nil
	}
	return resolved, false, nil
}

// CheckSymlinkResolutionWithinRoot は既存 path の symlink 解決後も root 配下かを判定する。
func CheckSymlinkResolutionWithinRoot(root, resolvedPath string) (outsideRoot bool, err error) {
	evaluated, err := filepath.EvalSymlinks(resolvedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		// 解決不能 path は lexical 判定で許可し、実行時に caller 側の失敗へ委ねる。
		return false, nil
	}

	ok, err := IsWithinRoot(root, filepath.Clean(evaluated))
	if err != nil {
		return false, fmt.Errorf("failed to resolve symlink for %q: %w", resolvedPath, err)
	}
	if !ok {
		return true, nil
	}
	return false, nil
}
