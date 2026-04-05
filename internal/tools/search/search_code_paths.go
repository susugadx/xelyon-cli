package search

import (
	"os"
	"path/filepath"
	"strings"
)

func collectFilePaths(results []SearchResult, opts SearchOptions) []string {
	paths := make([]string, 0, len(results))
	for _, r := range results {
		if r.FilePath == "" {
			continue
		}
		if absPath := absoluteAffectedFilePath(r.FilePath, opts, affectedFileSourceText); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}

func collectAffectedFilesFromExecutions(collected []formattedPatternExecution, opts SearchOptions) []string {
	paths := make([]string, 0, len(collected)*2)
	var outputs []string
	for _, execution := range collected {
		paths = append(paths, execution.AffectedFiles...)
		outputs = append(outputs, execution.Output)
	}
	paths = append(paths, collectPrimaryFilePathsFromOutputs(outputs, opts)...)
	return dedupePaths(paths)
}

func deriveAffectedFilesFromCachedResult(bundle *SymbolBundle, output string, opts SearchOptions) []string {
	if affected := collectSymbolBundleAffectedFiles(bundle, opts); len(affected) > 0 {
		return affected
	}
	return collectPrimaryFilePathsFromOutputs([]string{output}, opts)
}

func collectSymbolBundleAffectedFiles(bundle *SymbolBundle, opts SearchOptions) []string {
	if bundle == nil {
		return nil
	}

	paths := make([]string, 0, 1+len(bundle.Sections))
	rootPath := strings.TrimSpace(bundle.Debug.FileRootPath)
	add := func(file string) {
		if absPath := absoluteAffectedFilePathForSymbol(file, opts, rootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}

	add(bundle.Definition.File)
	for _, section := range bundle.Sections {
		for _, item := range section.Items {
			add(item.File)
		}
	}
	if bundle.Impact != nil {
		for _, item := range bundle.Impact.RecommendedReads {
			add(item.File)
		}
	}
	for _, file := range bundle.Debug.DependencyFiles {
		if absPath := absoluteAffectedFilePathWithBase(file, rootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}

func collectPrimaryFilePathsFromOutputs(outputs []string, opts SearchOptions) []string {
	var paths []string
	for _, output := range outputs {
		for _, file := range extractPrimaryFilePaths(output) {
			if absPath := absoluteAffectedFilePath(file, opts, affectedFileSourceText); absPath != "" {
				paths = append(paths, absPath)
			}
		}
	}
	return dedupePaths(paths)
}

type affectedFileSource int

const (
	affectedFileSourceText affectedFileSource = iota
	affectedFileSourceSymbol
)

func absoluteAffectedFilePath(file string, opts SearchOptions, source affectedFileSource) string {
	return absoluteAffectedFilePathWithBase(file, affectedFileBasePath(opts, source))
}

func absoluteAffectedFilePathForSymbol(file string, opts SearchOptions, rootPath string) string {
	rootPath = strings.TrimSpace(rootPath)
	if rootPath != "" {
		return absoluteAffectedFilePathWithBase(file, rootPath)
	}
	return absoluteAffectedFilePath(file, opts, affectedFileSourceSymbol)
}

func absoluteAffectedFilePathWithBase(file, basePath string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	if filepath.IsAbs(file) {
		return filepath.Clean(file)
	}

	basePath = strings.TrimSpace(basePath)
	if basePath != "" {
		return filepath.Clean(filepath.Join(basePath, filepath.FromSlash(file)))
	}

	if absPath, err := filepath.Abs(file); err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(file)
}

func affectedFileBasePath(opts SearchOptions, source affectedFileSource) string {
	switch source {
	case affectedFileSourceSymbol:
		if root := strings.TrimSpace(opts.ProjectMapRootPath); root != "" {
			if abs, err := filepath.Abs(root); err == nil {
				return abs
			}
			return filepath.Clean(root)
		}
	}
	return invocationCWDOrGetwd(opts)
}

func invocationCWDOrGetwd(opts SearchOptions) string {
	if cwd := strings.TrimSpace(opts.InvocationCWD); cwd != "" {
		if abs, err := filepath.Abs(cwd); err == nil {
			return abs
		}
		return filepath.Clean(cwd)
	}
	if cwd, err := os.Getwd(); err == nil {
		if abs, err := filepath.Abs(cwd); err == nil {
			return abs
		}
		return filepath.Clean(cwd)
	}
	return ""
}

func dedupePaths(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}
