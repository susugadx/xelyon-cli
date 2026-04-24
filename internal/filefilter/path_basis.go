package filefilter

import (
	"os"
	"path/filepath"
	"strings"
)

type SearchPathBasis struct {
	Workdir   string
	Target    string
	MatchRoot string
}

func MatchPath(filePath string, searchPath string) string {
	return MatchPathWithWorkspace(filePath, searchPath, "")
}

// MatchPathWithWorkspace は direct directory filter と search post-filter で共有する
// workspace-relative 基準へ候補 path を正規化する。
func MatchPathWithWorkspace(filePath string, searchPath string, workspaceRoot string) string {
	return WorkspaceRelativePath(filePath, MatchRootWithWorkspace(searchPath, workspaceRoot))
}

func MatchRootWithWorkspace(searchPath string, workspaceRoot string) string {
	return ResolveSearchPathBasisWithWorkspace(searchPath, workspaceRoot).MatchRoot
}

// WorkspaceRelativePath は file path を共有 file_filter matching 基準へ変換する。
// 絶対 path は workspace root 配下なら相対化し、相対 path はそのまま扱う。
func WorkspaceRelativePath(filePath string, workspaceRoot string) string {
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

func ResolveSearchPathBasis(searchPath string) SearchPathBasis {
	return ResolveSearchPathBasisWithWorkspace(searchPath, "")
}

// ResolveSearchPathBasisWithWorkspace は ripgrep/grep の実行範囲を指定 path に閉じつつ、
// workspace 配下の絶対 path では workspace-relative な file_filter 意味論を保つ。
func ResolveSearchPathBasisWithWorkspace(searchPath string, workspaceRoot string) SearchPathBasis {
	searchPath = strings.TrimSpace(searchPath)
	workspaceRoot = NormalizeWorkspaceRoot(workspaceRoot)
	if searchPath == "" {
		return SearchPathBasis{Target: "."}
	}

	cleanPath := filepath.Clean(searchPath)
	if !filepath.IsAbs(cleanPath) {
		return SearchPathBasis{Target: cleanPath}
	}

	if workspaceScopedBasis, ok := resolveWorkspaceScopedBasis(cleanPath, workspaceRoot); ok {
		return workspaceScopedBasis
	}
	if statBasis, ok := resolvePathBasisFromStat(cleanPath); ok {
		return statBasis
	}
	return SearchPathBasis{Target: cleanPath}
}

func resolveWorkspaceScopedBasis(cleanPath, workspaceRoot string) (SearchPathBasis, bool) {
	if workspaceRoot == "" {
		return SearchPathBasis{}, false
	}
	if cleanPath == workspaceRoot {
		return SearchPathBasis{
			Workdir:   workspaceRoot,
			Target:    ".",
			MatchRoot: workspaceRoot,
		}, true
	}
	relPath, ok := relativeSearchFileFilterPath(workspaceRoot, cleanPath)
	if !ok {
		return SearchPathBasis{}, false
	}
	return SearchPathBasis{
		Workdir:   workspaceRoot,
		Target:    relPath,
		MatchRoot: workspaceRoot,
	}, true
}

func resolvePathBasisFromStat(cleanPath string) (SearchPathBasis, bool) {
	info, err := os.Stat(cleanPath)
	if err != nil {
		return SearchPathBasis{}, false
	}
	if info.IsDir() {
		return SearchPathBasis{
			Workdir:   cleanPath,
			Target:    ".",
			MatchRoot: cleanPath,
		}, true
	}

	root := filepath.Dir(cleanPath)
	return SearchPathBasis{
		Workdir:   root,
		Target:    filepath.Base(cleanPath),
		MatchRoot: root,
	}, true
}

// ResolveWorkspaceRoot は file_filter matching の基準 workspace root を選ぶ。
func ResolveWorkspaceRoot(candidates ...string) string {
	for _, candidate := range candidates {
		if root := NormalizeWorkspaceRoot(candidate); root != "" {
			return root
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		return NormalizeWorkspaceRoot(cwd)
	}
	return ""
}

func NormalizeWorkspaceRoot(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if absPath, err := filepath.Abs(path); err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(path)
}
