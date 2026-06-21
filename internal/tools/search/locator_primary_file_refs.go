package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/search/internal/primaryrefs"
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

type primaryFileRefCandidateCollector struct {
	candidates []primaryFileRefCandidate
}

type primaryFileRefLineParser func(line string) (primaryFileRefCandidate, bool)

var primaryFileRefLineParsers = []primaryFileRefLineParser{
	parseTextPrimaryFileRefCandidate,
	parseStructuredHeaderPrimaryFileRefCandidate,
	parseNumberedPrimaryFileRefCandidate,
}

func extractPrimaryFilePaths(output string) []string {
	return primaryrefs.ExtractPaths(output)
}

func extractPrimaryFileRefs(output string, opts SearchOptions) []primaryFileRef {
	refs := primaryrefs.ExtractRefs(output, primaryFileRefResolverForOptions(opts))
	return primaryFileRefsFromPackageRefs(refs)
}

func collectPrimaryFileRefCandidates(output string) []primaryFileRefCandidate {
	return primaryFileRefCandidatesFromPackageCandidates(primaryrefs.CollectCandidates(output))
}

func parsePrimaryFileRefCandidateLine(line string) (primaryFileRefCandidate, bool) {
	candidate, ok := primaryrefs.ParseCandidateLine(line)
	if !ok {
		return primaryFileRefCandidate{}, false
	}
	return primaryFileRefCandidateFromPackageCandidate(candidate), true
}

func parseTextPrimaryFileRefCandidate(line string) (primaryFileRefCandidate, bool) {
	candidate, ok := primaryrefs.ParseCandidateLine(line)
	if !ok || candidate.Source != primaryrefs.SourceText {
		return primaryFileRefCandidate{}, false
	}
	return primaryFileRefCandidateFromPackageCandidate(candidate), true
}

func parseStructuredHeaderPrimaryFileRefCandidate(line string) (primaryFileRefCandidate, bool) {
	candidate, ok := primaryrefs.ParseCandidateLine(line)
	if !ok || candidate.Source != primaryrefs.SourceStructuredSymbol || !strings.HasPrefix(strings.TrimSpace(line), "── ") {
		return primaryFileRefCandidate{}, false
	}
	return primaryFileRefCandidateFromPackageCandidate(candidate), true
}

func parseNumberedPrimaryFileRefCandidate(line string) (primaryFileRefCandidate, bool) {
	candidate, ok := primaryrefs.ParseCandidateLine(line)
	if !ok || !primaryrefs.HasNumericListPrefix(line) {
		return primaryFileRefCandidate{}, false
	}
	return primaryFileRefCandidateFromPackageCandidate(candidate), true
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
	refs := primaryrefs.ResolveCandidates(primaryFileRefPackageCandidates(candidates), primaryFileRefResolverForOptions(opts))
	return primaryFileRefsFromPackageRefs(refs)
}

func primaryFileRefResolverForOptions(opts SearchOptions) primaryrefs.Resolver {
	return primaryrefs.Resolver{
		Text: func(displayPath string) string {
			return resolvePrimaryFileRefPath(displayPath, opts, primaryFileRefSourceText)
		},
		StructuredSymbol: func(displayPath string) string {
			return resolvePrimaryFileRefPath(displayPath, opts, primaryFileRefSourceStructuredSymbol)
		},
		InvocationRelative: func(displayPath string) string {
			return resolvePrimaryFileRefPath(displayPath, opts, primaryFileRefSourceInvocationRelative)
		},
	}
}

func primaryFileRefPackageCandidate(candidate primaryFileRefCandidate) primaryrefs.Candidate {
	return primaryrefs.Candidate{
		DisplayPath: candidate.DisplayPath,
		Source:      primaryFileRefPackageSource(candidate.Source),
	}
}

func primaryFileRefPackageCandidates(candidates []primaryFileRefCandidate) []primaryrefs.Candidate {
	packageCandidates := make([]primaryrefs.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		packageCandidates = append(packageCandidates, primaryFileRefPackageCandidate(candidate))
	}
	return packageCandidates
}

func primaryFileRefCandidateFromPackageCandidate(candidate primaryrefs.Candidate) primaryFileRefCandidate {
	return primaryFileRefCandidate{
		DisplayPath: candidate.DisplayPath,
		Source:      primaryFileRefSourceFromPackageSource(candidate.Source),
	}
}

func primaryFileRefCandidatesFromPackageCandidates(candidates []primaryrefs.Candidate) []primaryFileRefCandidate {
	searchCandidates := make([]primaryFileRefCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		searchCandidates = append(searchCandidates, primaryFileRefCandidateFromPackageCandidate(candidate))
	}
	return searchCandidates
}

func primaryFileRefPackageSource(source primaryFileRefSource) primaryrefs.Source {
	switch source {
	case primaryFileRefSourceStructuredSymbol:
		return primaryrefs.SourceStructuredSymbol
	case primaryFileRefSourceInvocationRelative:
		return primaryrefs.SourceInvocationRelative
	default:
		return primaryrefs.SourceText
	}
}

func primaryFileRefSourceFromPackageSource(source primaryrefs.Source) primaryFileRefSource {
	switch source {
	case primaryrefs.SourceStructuredSymbol:
		return primaryFileRefSourceStructuredSymbol
	case primaryrefs.SourceInvocationRelative:
		return primaryFileRefSourceInvocationRelative
	default:
		return primaryFileRefSourceText
	}
}

func primaryFileRefFromPackageRef(ref primaryrefs.Ref) primaryFileRef {
	return primaryFileRef{
		DisplayPath:  ref.DisplayPath,
		ResolvedPath: ref.ResolvedPath,
	}
}

func primaryFileRefsFromPackageRefs(refs []primaryrefs.Ref) []primaryFileRef {
	searchRefs := make([]primaryFileRef, 0, len(refs))
	for _, ref := range refs {
		searchRefs = append(searchRefs, primaryFileRefFromPackageRef(ref))
	}
	return searchRefs
}

func hasNumericListPrefix(line string) bool {
	return primaryrefs.HasNumericListPrefix(line)
}

func splitNumberedListLinePrefix(line string) (string, bool) {
	return primaryrefs.SplitNumberedListLinePrefix(line)
}

func parseNumberedCandidateFilePath(line string) (string, bool) {
	return primaryrefs.ParseNumberedCandidateFilePath(line)
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
