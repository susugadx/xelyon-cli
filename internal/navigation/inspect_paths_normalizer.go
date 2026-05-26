package navigation

import (
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

	sourceBase := resolveNavigationSourceBase(runtime.InvocationCWD)

	for i := range result.Callers {
		result.Callers[i].File = normalizeReferenceFilePath(result.Callers[i], targetRoot, sourceBase)
	}
	for i := range result.Refs {
		result.Refs[i].File = normalizeReferenceFilePath(result.Refs[i], targetRoot, sourceBase)
	}
	for i := range result.Tests {
		result.Tests[i].File = normalizeResolvedResultFilePath(result.Tests[i].File, result.Tests[i].ResolvedPath, targetRoot, sourceBase)
	}
	for i := range result.Implementations {
		result.Implementations[i].File = normalizeResolvedResultFilePath(result.Implementations[i].File, result.Implementations[i].ResolvedPath, targetRoot, sourceBase)
	}
}

func preferredInspectRootPath(symbolRoot, projectRoot string) string {
	symbolRoot = normalizeNavigationRootPath(symbolRoot)
	projectRoot = normalizeNavigationRootPath(projectRoot)

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

func pathWithinRoot(rootPath, candidatePath string) bool {
	rel, err := filepath.Rel(rootPath, candidatePath)
	if err != nil {
		return false
	}
	rel = filepath.Clean(rel)
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, "../"))
}

func normalizeResultFilePath(path, targetRoot, sourceBase string) string {
	return normalizeResolvedResultFilePath(path, "", targetRoot, sourceBase)
}

func normalizeReferenceFilePath(ref Reference, targetRoot, sourceBase string) string {
	return normalizeResolvedResultFilePath(ref.File, ref.ResolvedPath, targetRoot, sourceBase)
}

func normalizeResolvedResultFilePath(path, resolvedPath, targetRoot, sourceBase string) string {
	path = strings.TrimSpace(path)
	resolvedPath = strings.TrimSpace(resolvedPath)
	if path == "" && resolvedPath == "" {
		return ""
	}

	if resolvedPath != "" {
		if absPath, err := filepath.Abs(filepath.FromSlash(resolvedPath)); err == nil && pathExists(absPath) {
			if rel, ok := absoluteToSnapshotRel(targetRoot, absPath); ok {
				return filepath.Clean(filepath.ToSlash(rel))
			}
		}
	}

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
		sourceRelativeAbs := filepath.Join(sourceBase, filepath.FromSlash(path))
		if rel, ok := absoluteToSnapshotRel(targetRoot, sourceRelativeAbs); ok && pathExists(sourceRelativeAbs) {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	// 一部ヘルパーは process cwd 相対の path を返すため、存在確認付きで回収する。
	if absPath, err := filepath.Abs(filepath.FromSlash(path)); err == nil && pathExists(absPath) {
		if rel, ok := absoluteToSnapshotRel(targetRoot, absPath); ok {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	if sourceBase != "" {
		sourceRelativeAbs := filepath.Join(sourceBase, filepath.FromSlash(path))
		if rel, ok := absoluteToSnapshotRel(targetRoot, sourceRelativeAbs); ok {
			return filepath.Clean(filepath.ToSlash(rel))
		}
	}

	return filepath.Clean(filepath.ToSlash(path))
}
