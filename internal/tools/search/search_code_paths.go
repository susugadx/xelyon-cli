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
	for _, execution := range collected {
		paths = append(paths, execution.AffectedFiles...)
		paths = append(paths, collectPrimaryAffectedFilePathsFromOutput(execution.Output, opts)...)
	}
	return dedupePaths(paths)
}

func deriveAffectedFilesFromCachedResult(bundle *SymbolBundle, output string, opts SearchOptions) []string {
	if affected := collectSymbolBundleAffectedFiles(bundle, opts); len(affected) > 0 {
		return affected
	}
	return collectPrimaryAffectedFilePathsFromOutput(output, opts)
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

func collectPrimaryAffectedFilePathsFromOutput(output string, opts SearchOptions) []string {
	var paths []string
	seen := make(map[string]bool)
	add := func(file string, source affectedFileSource) {
		var absPath string
		switch source {
		case affectedFileSourceSymbol:
			absPath = absoluteAffectedFilePathForSymbol(file, opts, "")
		default:
			absPath = absoluteAffectedFilePath(file, opts, affectedFileSourceText)
		}
		if absPath == "" || seen[absPath] {
			return
		}
		seen[absPath] = true
		paths = append(paths, absPath)
	}

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "📄 ") {
			rest := strings.TrimPrefix(trimmed, "📄 ")
			if idx := strings.Index(rest, " ("); idx > 0 {
				add(rest[:idx], affectedFileSourceText)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "── ") && strings.Contains(trimmed, " in ") && strings.HasSuffix(trimmed, "──") {
			inIdx := strings.LastIndex(trimmed, " in ")
			rest := trimmed[inIdx+4:]
			rest = strings.TrimSuffix(rest, "──")
			rest = strings.TrimSpace(rest)
			if atIdx := strings.LastIndex(rest, " @"); atIdx > 0 {
				rest = rest[:atIdx]
			}
			add(rest, affectedFileSourceSymbol)
			continue
		}
		if hasNumericListPrefix(trimmed) {
			if numbered, ok := parseNumberedCandidateFilePath(trimmed); ok {
				add(numbered, affectedFileSourceSymbol)
				continue
			}
			if idx := strings.LastIndex(trimmed, " in "); idx > 0 {
				add(strings.TrimSpace(trimmed[idx+4:]), affectedFileSourceText)
				continue
			}
		}
	}
	return paths
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
	return absoluteAffectedFilePathWithPreferredBases(file, symbolAffectedFileBaseCandidates(opts, rootPath)...)
}

func absoluteAffectedFilePathWithBase(file, basePath string) string {
	return absoluteAffectedFilePathWithPreferredBases(file, basePath)
}

func absoluteAffectedFilePathWithPreferredBases(file string, basePaths ...string) string {
	file = strings.TrimSpace(file)
	if file == "" {
		return ""
	}
	if filepath.IsAbs(file) {
		return filepath.Clean(file)
	}

	var fallback string
	seen := make(map[string]bool, len(basePaths))
	for _, basePath := range basePaths {
		basePath = normalizeAffectedFileBase(basePath)
		if basePath == "" || seen[basePath] {
			continue
		}
		seen[basePath] = true
		candidate := filepath.Clean(filepath.Join(basePath, filepath.FromSlash(file)))
		if fallback == "" {
			fallback = candidate
		}
		if affectedFileExists(candidate) {
			return candidate
		}
	}

	if fallback != "" {
		return fallback
	}

	if absPath, err := filepath.Abs(filepath.FromSlash(file)); err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(file)
}

func normalizeAffectedFileBase(basePath string) string {
	basePath = strings.TrimSpace(basePath)
	if basePath == "" {
		return ""
	}
	if absPath, err := filepath.Abs(basePath); err == nil {
		return filepath.Clean(absPath)
	}
	return filepath.Clean(basePath)
}

func affectedFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func symbolAffectedFileBaseCandidates(opts SearchOptions, rootPath string) []string {
	bases := make([]string, 0, 4)
	if root := strings.TrimSpace(opts.ProjectMapRootPath); root != "" {
		bases = append(bases, root)
	}
	if rootPath = strings.TrimSpace(rootPath); rootPath != "" {
		bases = append(bases, rootPath)
	}
	if cwd := strings.TrimSpace(opts.InvocationCWD); cwd != "" {
		bases = append(bases, cwd)
	}
	if cwd := invocationCWDOrGetwd(opts); cwd != "" {
		bases = append(bases, cwd)
	}
	return bases
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
