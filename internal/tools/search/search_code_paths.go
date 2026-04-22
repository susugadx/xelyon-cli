package search

import (
	"os"
	"path/filepath"
	"strings"
)

type primaryAffectedPathCollector struct {
	seen  map[string]bool
	paths []string
}

type affectedFileCandidateResolution struct {
	Matched  string
	Fallback string
}

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
		paths = append(paths, affectedFilePathsFromExecution(execution, opts)...)
	}
	return dedupePaths(paths)
}

func deriveAffectedFilesFromCachedResult(bundle *SymbolBundle, output string, opts SearchOptions) []string {
	return affectedFilePathsFromBundleOrOutput(bundle, output, opts)
}

func affectedFilePathsFromExecution(execution formattedPatternExecution, opts SearchOptions) []string {
	paths := append([]string(nil), execution.AffectedFiles...)
	paths = append(paths, affectedFilePathsFromBundleOrOutput(execution.Bundle, execution.Output, opts)...)
	return dedupePaths(paths)
}

func affectedFilePathsFromBundleOrOutput(bundle *SymbolBundle, output string, opts SearchOptions) []string {
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
	paths = appendBundleFileAffectedPath(paths, bundle.Definition.File, opts, rootPath)
	paths = appendBundleSectionAffectedPaths(paths, bundle.Sections, opts, rootPath)
	paths = appendBundleImpactAffectedPaths(paths, bundle.Impact, opts, rootPath)
	paths = appendBundleDependencyAffectedPaths(paths, bundle.Debug.DependencyFiles, rootPath)
	return dedupePaths(paths)
}

func appendBundleSectionAffectedPaths(paths []string, sections []SymbolBundleSection, opts SearchOptions, rootPath string) []string {
	for _, section := range sections {
		for _, item := range section.Items {
			paths = appendBundleItemAffectedPath(paths, item, opts, rootPath)
		}
	}
	return paths
}

func appendBundleImpactAffectedPaths(paths []string, impact *SymbolBundleImpact, opts SearchOptions, rootPath string) []string {
	if impact == nil {
		return paths
	}
	for _, item := range impact.RecommendedReads {
		paths = appendBundleItemAffectedPath(paths, item, opts, rootPath)
	}
	return paths
}

func appendBundleDependencyAffectedPaths(paths []string, dependencyFiles []string, rootPath string) []string {
	for _, file := range dependencyFiles {
		if absPath := absoluteAffectedFilePathWithBase(file, rootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return paths
}

func appendBundleItemAffectedPath(paths []string, item SymbolBundleItem, opts SearchOptions, rootPath string) []string {
	if resolved := cleanResolvedLocatorPath(item.ResolvedPath); resolved != "" {
		return append(paths, resolved)
	}
	return appendBundleFileAffectedPath(paths, item.File, opts, rootPath)
}

func appendBundleFileAffectedPath(paths []string, file string, opts SearchOptions, rootPath string) []string {
	if absPath := absoluteAffectedFilePathForBundle(file, opts, rootPath); absPath != "" {
		return append(paths, absPath)
	}
	return paths
}

func collectPrimaryAffectedFilePathsFromOutput(output string, opts SearchOptions) []string {
	collector := newPrimaryAffectedPathCollector()
	for _, ref := range extractPrimaryFileRefs(output, opts) {
		collector.addRef(ref)
	}
	return collector.results()
}

func newPrimaryAffectedPathCollector() *primaryAffectedPathCollector {
	return &primaryAffectedPathCollector{
		seen: make(map[string]bool),
	}
}

func (collector *primaryAffectedPathCollector) addRef(ref primaryFileRef) {
	absPath := strings.TrimSpace(ref.ResolvedPath)
	if absPath == "" || collector.seen[absPath] {
		return
	}
	collector.seen[absPath] = true
	collector.paths = append(collector.paths, absPath)
}

func (collector *primaryAffectedPathCollector) results() []string {
	return collector.paths
}

type affectedFileSource int

const (
	affectedFileSourceText affectedFileSource = iota
	affectedFileSourceSymbol
)

type affectedFileBaseResolver func(opts SearchOptions) string

var affectedFileBaseResolvers = map[affectedFileSource]affectedFileBaseResolver{
	affectedFileSourceText:   resolveTextAffectedFileBase,
	affectedFileSourceSymbol: resolveSymbolAffectedFileBase,
}

func absoluteAffectedFilePath(file string, opts SearchOptions, source affectedFileSource) string {
	return absoluteAffectedFilePathWithBase(file, affectedFileBasePath(opts, source))
}

func absoluteAffectedFilePathForSymbol(file string, opts SearchOptions, rootPath string) string {
	return absoluteAffectedFilePathWithPreferredBases(file, symbolAffectedFileBaseCandidates(opts, rootPath)...)
}

func absoluteAffectedFilePathForBundle(file string, opts SearchOptions, rootPath string) string {
	return absoluteAffectedFilePathWithPreferredBases(file, bundleAffectedFileBaseCandidates(opts, rootPath)...)
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
	resolution := resolveAffectedFileCandidateFromBases(file, basePaths)
	if resolution.Matched != "" {
		return resolution.Matched
	}
	if resolution.Fallback != "" {
		return resolution.Fallback
	}
	return fallbackAbsoluteAffectedFilePath(file)
}

func resolveAffectedFileCandidateFromBases(file string, basePaths []string) affectedFileCandidateResolution {
	resolution := affectedFileCandidateResolution{}
	for _, basePath := range normalizedUniqueBasePaths(basePaths) {
		candidate := affectedFileCandidatePath(file, basePath)
		if resolution.Fallback == "" {
			resolution.Fallback = candidate
		}
		if affectedFileExists(candidate) {
			resolution.Matched = candidate
			return resolution
		}
	}
	return resolution
}

func normalizedUniqueBasePaths(basePaths []string) []string {
	seen := make(map[string]bool, len(basePaths))
	normalized := make([]string, 0, len(basePaths))
	for _, basePath := range basePaths {
		basePath = normalizeAffectedFileBase(basePath)
		if basePath == "" || seen[basePath] {
			continue
		}
		seen[basePath] = true
		normalized = append(normalized, basePath)
	}
	return normalized
}

func affectedFileCandidatePath(file, basePath string) string {
	return filepath.Clean(filepath.Join(basePath, filepath.FromSlash(file)))
}

func fallbackAbsoluteAffectedFilePath(file string) string {
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
	return affectedFileBaseCandidates(opts, opts.ProjectMapRootPath, rootPath)
}

func bundleAffectedFileBaseCandidates(opts SearchOptions, rootPath string) []string {
	return affectedFileBaseCandidates(opts, rootPath, opts.ProjectMapRootPath)
}

func affectedFileBaseCandidates(opts SearchOptions, prioritized ...string) []string {
	bases := make([]string, 0, len(prioritized)+2)
	for _, base := range prioritized {
		if base = strings.TrimSpace(base); base != "" {
			bases = append(bases, base)
		}
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
	if resolver, ok := affectedFileBaseResolvers[source]; ok {
		if root := normalizeAffectedFileBase(resolver(opts)); root != "" {
			return root
		}
	}
	return invocationCWDOrGetwd(opts)
}

func resolveSymbolAffectedFileBase(opts SearchOptions) string {
	return opts.ProjectMapRootPath
}

func resolveTextAffectedFileBase(opts SearchOptions) string {
	return searchFileFilterMatchRootWithWorkspace(opts.Path, resolveSearchWorkspaceRoot(opts))
}

func invocationCWDOrGetwd(opts SearchOptions) string {
	if cwd := normalizeAffectedFileBase(opts.InvocationCWD); cwd != "" {
		return cwd
	}
	if cwd, err := os.Getwd(); err == nil {
		return normalizeAffectedFileBase(cwd)
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
