package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

type primaryFileRef struct {
	DisplayPath  string
	ResolvedPath string
}

type primaryFileRefSource int

const (
	primaryFileRefSourceText primaryFileRefSource = iota
	primaryFileRefSourceStructuredSymbol
	primaryFileRefSourceInvocationRelative
)

type primaryFileRefCandidate struct {
	DisplayPath string
	Source      primaryFileRefSource
}

type primaryFileRefResolver struct {
	opts SearchOptions
	seen map[string]bool
	refs []primaryFileRef
}

type primaryFileRefCandidateCollector struct {
	candidates []primaryFileRefCandidate
}

type primaryFileRefLineParser func(line string) (primaryFileRefCandidate, bool)

var numberedCandidateKindTokens = []string{
	" function ",
	" method ",
	" type ",
	" interface ",
	" const ",
	" var ",
}

var primaryFileRefLineParsers = []primaryFileRefLineParser{
	parseTextPrimaryFileRefCandidate,
	parseStructuredHeaderPrimaryFileRefCandidate,
	parseNumberedPrimaryFileRefCandidate,
}

func newSearchLocator(displayPath, resolvedPath string, line, endLine int, name string) locator.Location {
	return locator.Location{
		FilePath:     displayPath,
		ResolvedPath: cleanResolvedLocatorPath(resolvedPath),
		Line:         line,
		EndLine:      endLine,
		Name:         name,
	}
}

func newTextSearchLocator(displayPath string, line, endLine int, name string, opts SearchOptions) locator.Location {
	return newSearchLocator(displayPath, absoluteAffectedFilePath(displayPath, opts, affectedFileSourceText), line, endLine, name)
}

func newBundleLocator(displayPath string, line, endLine int, name string, bundle *SymbolBundle) locator.Location {
	return newBundleScopedLocator(displayPath, "", line, endLine, name, bundle)
}

func newBundleItemLocator(item SymbolBundleItem, bundle *SymbolBundle) locator.Location {
	return newBundleScopedLocator(item.File, item.ResolvedPath, item.Line, item.EndLine, item.Name, bundle)
}

func newBundleScopedLocator(displayPath, resolvedPath string, line, endLine int, name string, bundle *SymbolBundle) locator.Location {
	if clean := cleanResolvedLocatorPath(resolvedPath); clean != "" {
		return newSearchLocator(displayPath, clean, line, endLine, name)
	}
	rootPath := ""
	if bundle != nil {
		rootPath = bundle.Debug.FileRootPath
	}
	return newSearchLocator(displayPath, absoluteAffectedFilePathWithBase(displayPath, rootPath), line, endLine, name)
}

func extractPrimaryFilePaths(output string) []string {
	refs := extractPrimaryFileRefs(output, SearchOptions{})
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		paths = append(paths, ref.DisplayPath)
	}
	return paths
}

func extractPrimaryFileRefs(output string, opts SearchOptions) []primaryFileRef {
	candidates := collectPrimaryFileRefCandidates(output)
	return resolvePrimaryFileRefs(candidates, opts)
}

func collectPrimaryFileRefCandidates(output string) []primaryFileRefCandidate {
	collector := newPrimaryFileRefCandidateCollector()
	for _, line := range strings.Split(output, "\n") {
		collector.addLine(line)
	}
	return collector.results()
}

func parsePrimaryFileRefCandidateLine(line string) (primaryFileRefCandidate, bool) {
	trimmed := strings.TrimSpace(line)
	for _, parser := range primaryFileRefLineParsers {
		if candidate, ok := parser(trimmed); ok {
			return candidate, true
		}
	}
	return primaryFileRefCandidate{}, false
}

func parseTextPrimaryFileRefCandidate(line string) (primaryFileRefCandidate, bool) {
	if !strings.HasPrefix(line, "📄 ") {
		return primaryFileRefCandidate{}, false
	}
	rest := strings.TrimPrefix(line, "📄 ")
	idx := strings.Index(rest, " (")
	if idx <= 0 {
		return primaryFileRefCandidate{}, false
	}
	return primaryFileRefCandidate{
		DisplayPath: strings.TrimSpace(rest[:idx]),
		Source:      primaryFileRefSourceText,
	}, true
}

func parseStructuredHeaderPrimaryFileRefCandidate(line string) (primaryFileRefCandidate, bool) {
	if !strings.HasPrefix(line, "── ") || !strings.Contains(line, " in ") || !strings.HasSuffix(line, "──") {
		return primaryFileRefCandidate{}, false
	}
	inIdx := strings.LastIndex(line, " in ")
	rest := line[inIdx+4:]
	rest = strings.TrimSuffix(rest, "──")
	return primaryFileRefCandidate{
		DisplayPath: trimRenderedPrimaryFilePath(rest),
		Source:      primaryFileRefSourceStructuredSymbol,
	}, true
}

func parseNumberedPrimaryFileRefCandidate(line string) (primaryFileRefCandidate, bool) {
	rest, ok := splitNumberedListLinePrefix(line)
	if !ok {
		return primaryFileRefCandidate{}, false
	}
	if numbered, ok := parseNumberedCandidateFilePathWithPolicy(rest, numberedCandidateKindTokens); ok {
		return primaryFileRefCandidate{
			DisplayPath: trimRenderedPrimaryFilePath(numbered),
			Source:      primaryFileRefSourceStructuredSymbol,
		}, true
	}
	if idx := strings.LastIndex(rest, " in "); idx > 0 {
		return primaryFileRefCandidate{
			DisplayPath: trimRenderedPrimaryFilePath(rest[idx+4:]),
			Source:      primaryFileRefSourceInvocationRelative,
		}, true
	}
	return primaryFileRefCandidate{}, false
}

func newPrimaryFileRefCandidateCollector() *primaryFileRefCandidateCollector {
	return &primaryFileRefCandidateCollector{
		candidates: make([]primaryFileRefCandidate, 0),
	}
}

func (collector *primaryFileRefCandidateCollector) addLine(line string) {
	candidate, ok := parsePrimaryFileRefCandidateLine(line)
	if !ok {
		return
	}
	collector.addCandidate(candidate)
}

func (collector *primaryFileRefCandidateCollector) addCandidate(candidate primaryFileRefCandidate) {
	candidate.DisplayPath = strings.TrimSpace(candidate.DisplayPath)
	if candidate.DisplayPath == "" {
		return
	}
	collector.candidates = append(collector.candidates, candidate)
}

func (collector *primaryFileRefCandidateCollector) results() []primaryFileRefCandidate {
	return collector.candidates
}

func resolvePrimaryFileRefs(candidates []primaryFileRefCandidate, opts SearchOptions) []primaryFileRef {
	resolver := newPrimaryFileRefResolver(opts)
	for _, candidate := range candidates {
		resolver.addCandidate(candidate)
	}
	return resolver.results()
}

func newPrimaryFileRefResolver(opts SearchOptions) *primaryFileRefResolver {
	return &primaryFileRefResolver{
		opts: opts,
		seen: make(map[string]bool),
	}
}

func (resolver *primaryFileRefResolver) addCandidate(candidate primaryFileRefCandidate) {
	ref, ok := resolver.resolveCandidate(candidate)
	if !ok {
		return
	}
	key := primaryFileRefKey(ref)
	if resolver.seen[key] {
		return
	}
	resolver.seen[key] = true
	resolver.refs = append(resolver.refs, ref)
}

func (resolver *primaryFileRefResolver) resolveCandidate(candidate primaryFileRefCandidate) (primaryFileRef, bool) {
	displayPath := strings.TrimSpace(candidate.DisplayPath)
	if displayPath == "" {
		return primaryFileRef{}, false
	}
	return primaryFileRef{
		DisplayPath:  displayPath,
		ResolvedPath: cleanResolvedLocatorPath(resolvePrimaryFileRefPath(displayPath, resolver.opts, candidate.Source)),
	}, true
}

func (resolver *primaryFileRefResolver) results() []primaryFileRef {
	return resolver.refs
}

func primaryFileRefKey(ref primaryFileRef) string {
	return ref.DisplayPath + "\x00" + ref.ResolvedPath
}

func hasNumericListPrefix(line string) bool {
	_, ok := splitNumberedListLinePrefix(line)
	return ok
}

func splitNumberedListLinePrefix(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return "", false
	}
	for _, r := range line[:dotIdx] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	rest := strings.TrimSpace(line[dotIdx+1:])
	if rest == "" {
		return "", false
	}
	return rest, true
}

func parseNumberedCandidateFilePath(line string) (string, bool) {
	rest, ok := splitNumberedListLinePrefix(line)
	if !ok {
		return "", false
	}
	return parseNumberedCandidateFilePathWithPolicy(rest, numberedCandidateKindTokens)
}

func parseNumberedCandidateFilePathWithPolicy(rest string, kindTokens []string) (string, bool) {
	for _, token := range kindTokens {
		if idx := strings.Index(rest, token); idx > 0 {
			return strings.TrimSpace(rest[:idx]), true
		}
	}
	return "", false
}

func resolvePrimaryFileRefPath(displayPath string, opts SearchOptions, source primaryFileRefSource) string {
	switch source {
	case primaryFileRefSourceStructuredSymbol:
		return absoluteAffectedFilePathForSymbol(displayPath, opts, "")
	case primaryFileRefSourceInvocationRelative:
		return absoluteAffectedFilePathWithBase(displayPath, invocationCWDOrGetwd(opts))
	default:
		return absoluteAffectedFilePath(displayPath, opts, affectedFileSourceText)
	}
}

func cleanResolvedLocatorPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		path = abs
	}
	return filepath.Clean(path)
}

func trimRenderedPrimaryFilePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if atIdx := strings.LastIndex(path, " @"); atIdx > 0 {
		path = strings.TrimSpace(path[:atIdx])
	}
	for {
		lIdx := strings.LastIndex(path, " [L")
		if lIdx <= 0 || !strings.HasSuffix(path, "]") {
			break
		}
		path = strings.TrimSpace(path[:lIdx])
	}
	return path
}
