package search

import (
	"os"
	"path/filepath"
	"strings"
)

type searchPathBasis struct {
	workdir   string
	target    string
	matchRoot string
}

func searchFileFilterMatchPath(filePath string, searchPath string) string {
	return searchFileFilterMatchPathWithWorkspace(filePath, searchPath, "")
}

// searchFileFilterMatchPathWithWorkspace normalizes candidate paths onto the
// shared workspace-relative basis used by direct directory filters and search
// post-filters.
func searchFileFilterMatchPathWithWorkspace(filePath string, searchPath string, workspaceRoot string) string {
	return WorkspaceRelativeFileFilterPath(filePath, searchFileFilterMatchRootWithWorkspace(searchPath, workspaceRoot))
}

func searchFileFilterMatchRootWithWorkspace(searchPath string, workspaceRoot string) string {
	return resolveSearchPathBasisWithWorkspace(searchPath, workspaceRoot).matchRoot
}

// WorkspaceRelativeFileFilterPath converts file paths to the shared file_filter
// matching basis. Absolute paths are relativized to the workspace root when
// possible; relative paths are preserved as workspace-relative display paths.
func WorkspaceRelativeFileFilterPath(filePath string, workspaceRoot string) string {
	cleanPath := cleanFileFilterPath(filePath)
	if cleanPath == "" {
		return ""
	}
	if !filepath.IsAbs(filepath.Clean(cleanPath)) {
		return cleanPath
	}

	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		return cleanPath
	}

	relPath, ok := relativeSearchFileFilterPath(workspaceRoot, cleanPath)
	if !ok {
		return cleanPath
	}
	return relPath
}

func relativeSearchFileFilterPath(rootPath, filePath string) (string, bool) {
	rootPath = filepath.Clean(rootPath)
	filePath = filepath.Clean(filePath)

	relPath, err := filepath.Rel(rootPath, filePath)
	if err != nil {
		return "", false
	}
	relPath = filepath.Clean(relPath)
	if relPath == "." || relPath == "" || relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(relPath), true
}

func resolveSearchPathBasis(searchPath string) searchPathBasis {
	return resolveSearchPathBasisWithWorkspace(searchPath, "")
}

func resolveSearchPathBasisForOptions(opts SearchOptions) searchPathBasis {
	return resolveSearchPathBasisWithWorkspace(opts.Path, resolveSearchWorkspaceRoot(opts))
}

// resolveSearchPathBasisWithWorkspace keeps ripgrep/grep execution scoped to the
// requested path while preserving workspace-relative file_filter semantics when
// the path is an absolute path inside the current workspace.
func resolveSearchPathBasisWithWorkspace(searchPath string, workspaceRoot string) searchPathBasis {
	searchPath = strings.TrimSpace(searchPath)
	workspaceRoot = normalizeSearchWorkspaceRoot(workspaceRoot)
	if searchPath == "" {
		return searchPathBasis{target: "."}
	}

	cleanPath := filepath.Clean(searchPath)
	if !filepath.IsAbs(cleanPath) {
		return searchPathBasis{target: cleanPath}
	}

	if workspaceScopedBasis, ok := resolveWorkspaceScopedSearchBasis(cleanPath, workspaceRoot); ok {
		return workspaceScopedBasis
	}
	if statBasis, ok := resolveSearchBasisFromStat(cleanPath); ok {
		return statBasis
	}
	return searchPathBasis{target: cleanPath}
}

func resolveWorkspaceScopedSearchBasis(cleanPath, workspaceRoot string) (searchPathBasis, bool) {
	if workspaceRoot == "" {
		return searchPathBasis{}, false
	}
	if cleanPath == workspaceRoot {
		return searchPathBasis{
			workdir:   workspaceRoot,
			target:    ".",
			matchRoot: workspaceRoot,
		}, true
	}
	relPath, ok := relativeSearchFileFilterPath(workspaceRoot, cleanPath)
	if !ok {
		return searchPathBasis{}, false
	}
	return searchPathBasis{
		workdir:   workspaceRoot,
		target:    relPath,
		matchRoot: workspaceRoot,
	}, true
}

func resolveSearchBasisFromStat(cleanPath string) (searchPathBasis, bool) {
	info, err := os.Stat(cleanPath)
	if err != nil {
		return searchPathBasis{}, false
	}
	if info.IsDir() {
		return searchPathBasis{
			workdir:   cleanPath,
			target:    ".",
			matchRoot: cleanPath,
		}, true
	}

	root := filepath.Dir(cleanPath)
	return searchPathBasis{
		workdir:   root,
		target:    filepath.Base(cleanPath),
		matchRoot: root,
	}, true
}

func resolveSearchWorkspaceRoot(opts SearchOptions) string {
	for _, candidate := range []string{
		opts.ProjectMapRootPath,
		opts.InvocationCWD,
	} {
		if root := normalizeSearchWorkspaceRoot(candidate); root != "" {
			return root
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		return normalizeSearchWorkspaceRoot(cwd)
	}
	return ""
}

func normalizeSearchWorkspaceRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if absPath, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(path)
}
