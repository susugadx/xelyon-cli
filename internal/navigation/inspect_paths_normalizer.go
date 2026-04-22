package navigation

import (
	"os"
	"path/filepath"
	"strings"
)

func normalizeInspectResultPaths(result *InspectResult, runtime GoSymbolRuntime) {
	if result == nil || result.Symbol == nil {
		return
	}

	targetRoot := preferredInspectRootPath(result.Symbol.RootPath, runtime.ProjectMapRootPath)
	if targetRoot == "" {
		return
	}
	result.Symbol.RootPath = targetRoot

	sourceBase := strings.TrimSpace(runtime.InvocationCWD)
	if sourceBase == "" {
		if cwd, err := os.Getwd(); err == nil {
			sourceBase = cwd
		}
	}
	if sourceBase != "" {
		if abs, err := filepath.Abs(sourceBase); err == nil {
			sourceBase = abs
		}
	}

	for i := range result.Callers {
		result.Callers[i].File = normalizeResultFilePath(result.Callers[i].File, targetRoot, sourceBase)
	}
	for i := range result.Refs {
		result.Refs[i].File = normalizeResultFilePath(result.Refs[i].File, targetRoot, sourceBase)
	}
	for i := range result.Tests {
		result.Tests[i].File = normalizeResultFilePath(result.Tests[i].File, targetRoot, sourceBase)
	}
	for i := range result.Implementations {
		result.Implementations[i].File = normalizeResultFilePath(result.Implementations[i].File, targetRoot, sourceBase)
	}
}

func preferredInspectRootPath(symbolRoot, projectRoot string) string {
	symbolRoot = normalizeInspectRootPath(symbolRoot)
	projectRoot = normalizeInspectRootPath(projectRoot)

	switch {
	case symbolRoot == "":
		return projectRoot
	case projectRoot == "":
		return symbolRoot
	case pathWithinRoot(symbolRoot, projectRoot):
		return projectRoot
	case pathWithinRoot(projectRoot, symbolRoot):
		return symbolRoot
	default:
		return projectRoot
	}
}

func normalizeInspectRootPath(rootPath string) string {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return ""
	}
	if abs, err := filepath.Abs(rootPath); err == nil {
		return abs
	}
	return filepath.Clean(rootPath)
}

func pathWithinRoot(rootPath, candidatePath string) bool {
	rel, err := filepath.Rel(rootPath, candidatePath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}

func normalizeResultFilePath(path, targetRoot, sourceBase string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if filepath.IsAbs(path) {
		if rel, ok := absoluteToSnapshotRel(targetRoot, path); ok {
			return filepath.Clean(filepath.ToSlash(rel))
		}
		return filepath.Clean(path)
	}

	if targetRoot != "" {
		rootRelativeAbs := filepath.Join(targetRoot, filepath.FromSlash(path))
		if rel, ok := absoluteToSnapshotRel(targetRoot, rootRelativeAbs); ok && pathExists(rootRelativeAbs) {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	if sourceBase != "" {
		if rel, ok := absoluteToSnapshotRel(targetRoot, filepath.Join(sourceBase, filepath.FromSlash(path))); ok {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	// 一部ヘルパーは process cwd 相対の path を返すため、存在確認付きで回収する。
	if absPath, err := filepath.Abs(filepath.FromSlash(path)); err == nil && pathExists(absPath) {
		if rel, ok := absoluteToSnapshotRel(targetRoot, absPath); ok {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	return filepath.Clean(filepath.ToSlash(path))
}
