package search

import (
	"os"
	"path/filepath"
	"strings"
)

// structuredImpactSemanticReferenceFilterOptions は LSP など definition との
// 関係を検証できる collector の採用 scope を返す。direct file search だけは
// workspace refs を採用できるように広げ、directory / glob scope は維持する。
func structuredImpactSemanticReferenceFilterOptions(opts SearchOptions) SearchOptions {
	filter := opts
	if !structuredImpactSearchPathIsFile(opts) {
		return filter
	}

	root := strings.TrimSpace(structuredImpactWorkspaceRoot(opts))
	if root == "" {
		root = "."
	}
	filter.Path = root
	if root != "." && strings.TrimSpace(filter.ProjectMapRootPath) == "" {
		filter.ProjectMapRootPath = root
	}
	return filter
}

// structuredImpactNameOnlyEvidenceOptions は rg / test-name probe など
// 名前一致だけの evidence を、direct file search では definition 近辺に閉じる。
func structuredImpactNameOnlyEvidenceOptions(defFile string, opts SearchOptions) SearchOptions {
	nameOnly := opts
	if structuredImpactSearchPathIsFile(opts) {
		if dir := structuredImpactDefinitionDir(defFile, opts); dir != "" {
			nameOnly.Path = dir
		}
	}
	return nameOnly
}

func structuredImpactEvidenceFileTypeOptions(opts SearchOptions, fileType string) SearchOptions {
	if strings.TrimSpace(opts.FilePattern) != "" {
		opts.FileType = ""
		return opts
	}
	opts.FileType = strings.TrimSpace(fileType)
	return opts
}

func structuredImpactSearchPathIsFile(opts SearchOptions) bool {
	targetPath := searchTargetPathForOptions(opts)
	if targetPath == "" {
		return false
	}
	info, err := os.Stat(targetPath)
	return err == nil && !info.IsDir()
}

func structuredImpactDefinitionDir(defFile string, opts SearchOptions) string {
	defFile = strings.TrimSpace(defFile)
	if defFile == "" {
		target := searchTargetPathForOptions(opts)
		if target == "" {
			return ""
		}
		return filepath.Dir(target)
	}

	root := structuredImpactWorkspaceRoot(opts)
	if root == "" {
		root = invocationCWDOrGetwd(opts)
	}
	if root == "" {
		return ""
	}

	absPath := defFile
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(root, filepath.FromSlash(defFile))
	}
	return filepath.Dir(filepath.Clean(absPath))
}

func structuredImpactWorkspaceRoot(opts SearchOptions) string {
	basis := resolveSearchPathBasisForOptions(opts)
	if strings.TrimSpace(basis.MatchRoot) != "" {
		return basis.MatchRoot
	}
	if root := resolveSearchWorkspaceRoot(opts); strings.TrimSpace(root) != "" {
		return root
	}
	return "."
}
