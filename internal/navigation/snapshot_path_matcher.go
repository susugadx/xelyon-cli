package navigation

import (
	"os"
	"path/filepath"
	"strings"
)

// buildSnapshotPathMatcher は snapshot 相対パスに対する pathHint フィルタを構築する。
func buildSnapshotPathMatcher(rootPath, invocationCWD, pathHint string) func(string) bool {
	pathHint = strings.TrimSpace(pathHint)
	if pathHint == "" {
		return nil
	}

	root := strings.TrimSpace(rootPath)
	if root != "" {
		if abs, err := filepath.Abs(root); err == nil {
			root = abs
		}
	}

	normalizedHint := filepath.Clean(pathHint)
	absHint := normalizedHint
	if !filepath.IsAbs(absHint) {
		baseCWD := strings.TrimSpace(invocationCWD)
		if baseCWD == "" {
			if cwd, err := os.Getwd(); err == nil {
				baseCWD = cwd
			}
		}
		if baseCWD != "" {
			absHint = filepath.Join(baseCWD, normalizedHint)
		} else if abs, err := filepath.Abs(normalizedHint); err == nil {
			absHint = abs
		}
	}

	isDir := false
	if info, err := os.Stat(absHint); err == nil {
		isDir = info.IsDir()
	} else if filepath.Ext(normalizedHint) == "" {
		isDir = true
	}

	if root != "" {
		if isDir && filepath.Clean(absHint) == filepath.Clean(root) {
			return nil
		}
		if rel, ok := absoluteToSnapshotRel(root, absHint); ok {
			rel = filepath.Clean(filepath.ToSlash(rel))
			if isDir {
				return func(candidate string) bool {
					candidate = filepath.Clean(filepath.ToSlash(candidate))
					return candidate == rel || strings.HasPrefix(candidate, rel+"/")
				}
			}
			return func(candidate string) bool {
				return filepath.Clean(filepath.ToSlash(candidate)) == rel
			}
		}
	}

	cleanHint := filepath.Clean(filepath.ToSlash(normalizedHint))
	if isDir {
		return func(candidate string) bool {
			candidate = filepath.Clean(filepath.ToSlash(candidate))
			return candidate == cleanHint || strings.HasPrefix(candidate, cleanHint+"/")
		}
	}
	return func(candidate string) bool {
		return filepath.Clean(filepath.ToSlash(candidate)) == cleanHint
	}
}

// absoluteToSnapshotRel は rootPath 配下にある絶対パスを snapshot 相対パスへ変換する。
func absoluteToSnapshotRel(rootPath, absPath string) (string, bool) {
	if rootPath == "" || absPath == "" {
		return "", false
	}
	root := filepath.Clean(rootPath)
	abs := filepath.Clean(absPath)
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", false
	}
	rel = filepath.Clean(filepath.ToSlash(rel))
	if rel == "." || rel == "" || strings.HasPrefix(rel, "../") {
		return "", false
	}
	return rel, true
}
