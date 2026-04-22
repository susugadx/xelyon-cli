package navigation

import (
	"os"
	"path/filepath"
	"strings"
)

// toRelativePath は絶対パスを作業ディレクトリからの相対パスに変換する。
func toRelativePath(absPath string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return absPath
	}
	rel, err := filepath.Rel(cwd, absPath)
	if err != nil {
		return absPath
	}
	return rel
}

// cleanNavigationResolvedPath は navigation で扱う resolved path を正規化する。
func cleanNavigationResolvedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

// cleanInspectResolvedPath は inspect 表示向けの resolved path 正規化を行う。
func cleanInspectResolvedPath(path string) string {
	return cleanNavigationResolvedPath(path)
}

func normalizeNavigationBasePath(base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		return ""
	}
	if abs, err := filepath.Abs(base); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(base)
}

func normalizeNavigationRootPath(rootPath string) string {
	return normalizeNavigationBasePath(rootPath)
}

func resolveNavigationSourceBase(sourceBase string) string {
	base := normalizeNavigationBasePath(sourceBase)
	if base != "" {
		return base
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return normalizeNavigationBasePath(cwd)
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func cleanRelativeNavigationPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(filepath.FromSlash(path))
}

func resolveRelativePathAgainstBase(base, relativePath string) (string, bool) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", false
	}
	return filepath.Join(base, relativePath), true
}

func resolveExistingRelativePath(base, relativePath string) (string, bool) {
	candidate, ok := resolveRelativePathAgainstBase(base, relativePath)
	if !ok || !pathExists(candidate) {
		return "", false
	}
	return candidate, true
}

func resolveRelativePathFromPreferredBases(relativePath string, bases ...string) string {
	for _, base := range bases {
		if candidate, ok := resolveRelativePathAgainstBase(base, relativePath); ok {
			return candidate
		}
	}
	return relativePath
}

func resolveNavigationRelativeFilePath(file, invocationCWD, rootPath string) string {
	file = strings.TrimSpace(file)
	if file == "" || filepath.IsAbs(file) {
		return file
	}
	file = cleanRelativeNavigationPath(file)
	invocationBase := normalizeNavigationBasePath(invocationCWD)
	rootBase := normalizeNavigationRootPath(rootPath)
	if resolved, ok := resolveExistingRelativePath(invocationBase, file); ok {
		return resolved
	}
	if resolved, ok := resolveExistingRelativePath(rootBase, file); ok {
		return resolved
	}
	return resolveRelativePathFromPreferredBases(file, invocationBase, rootBase)
}
