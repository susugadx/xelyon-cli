package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func filterStructuredGoImpactEvidence(result navigation.InspectResult, opts SearchOptions) navigation.InspectResult {
	opts = structuredImpactSemanticReferenceFilterOptions(opts)
	if !structuredGoImpactEvidenceNeedsFilter(opts) {
		return result
	}

	var changed bool
	result.Callers, changed = filterStructuredGoImpactReferencesByScope(result.Callers, opts)
	if changed {
		result.TotalCallers = len(result.Callers)
		result.MoreCallers = false
	}
	result.Refs, changed = filterStructuredGoImpactReferencesByScope(result.Refs, opts)
	if changed {
		result.TotalRefs = len(result.Refs)
		result.MoreRefs = false
	}
	result.Tests, changed = filterStructuredGoImpactTestsByScope(result.Tests, opts)
	if changed {
		result.TotalTests = len(result.Tests)
		result.MoreTests = false
	}
	result.Implementations, _ = filterStructuredGoImpactImplementationsByScope(result.Implementations, opts)
	return result
}

func structuredGoImpactReferenceFilter(opts SearchOptions) navigation.ReferenceFilter {
	opts = structuredImpactSemanticReferenceFilterOptions(opts)
	if !structuredGoImpactEvidenceNeedsFilter(opts) {
		return nil
	}
	return func(ref navigation.Reference) bool {
		return structuredGoImpactEvidenceFileInScope(ref.File, ref.ResolvedPath, opts)
	}
}

func structuredGoImpactEvidenceNeedsFilter(opts SearchOptions) bool {
	return strings.TrimSpace(opts.Path) != "" ||
		strings.TrimSpace(opts.FileType) != "" ||
		strings.TrimSpace(opts.FilePattern) != "" ||
		opts.ignoreMatcher != nil
}

func filterStructuredGoImpactReferencesByScope(refs []navigation.Reference, opts SearchOptions) ([]navigation.Reference, bool) {
	if len(refs) == 0 {
		return refs, false
	}
	filtered := make([]navigation.Reference, 0, len(refs))
	for _, ref := range refs {
		if structuredGoImpactEvidenceFileInScope(ref.File, ref.ResolvedPath, opts) {
			filtered = append(filtered, ref)
		}
	}
	return filtered, len(filtered) != len(refs)
}

func filterStructuredGoImpactTestsByScope(tests []navigation.TestRef, opts SearchOptions) ([]navigation.TestRef, bool) {
	if len(tests) == 0 {
		return tests, false
	}
	filtered := make([]navigation.TestRef, 0, len(tests))
	for _, test := range tests {
		if structuredGoImpactEvidenceFileInScope(test.File, test.ResolvedPath, opts) {
			filtered = append(filtered, test)
		}
	}
	return filtered, len(filtered) != len(tests)
}

func filterStructuredGoImpactImplementationsByScope(impls []navigation.ImplementationRef, opts SearchOptions) ([]navigation.ImplementationRef, bool) {
	if len(impls) == 0 {
		return impls, false
	}
	filtered := make([]navigation.ImplementationRef, 0, len(impls))
	for _, impl := range impls {
		if structuredGoImpactEvidenceFileInScope(impl.File, impl.ResolvedPath, opts) {
			filtered = append(filtered, impl)
		}
	}
	return filtered, len(filtered) != len(impls)
}

func structuredGoImpactEvidenceFileInScope(file, resolvedPath string, opts SearchOptions) bool {
	candidate, ok := structuredGoImpactEvidenceCandidateForScope(file, resolvedPath, opts)
	if !ok {
		return false
	}
	for _, displayPath := range candidate.displayPaths {
		if searchCandidateAllowedByOptions(candidate.absPath, displayPath, opts) {
			return true
		}
	}
	return false
}

type structuredGoImpactEvidenceCandidate struct {
	absPath      string
	displayPaths []string
}

func structuredGoImpactEvidenceCandidateForScope(file, resolvedPath string, opts SearchOptions) (structuredGoImpactEvidenceCandidate, bool) {
	displayPath := strings.TrimSpace(file)
	pathCandidate := strings.TrimSpace(resolvedPath)
	if pathCandidate == "" {
		pathCandidate = displayPath
	}
	if pathCandidate == "" {
		return structuredGoImpactEvidenceCandidate{}, false
	}
	if displayPath == "" {
		displayPath = pathCandidate
	}
	absPath := structuredGoImpactEvidenceAbsolutePath(pathCandidate, opts)
	return structuredGoImpactEvidenceCandidate{
		absPath:      absPath,
		displayPaths: structuredGoImpactEvidenceDisplayPathCandidates(absPath, displayPath, opts),
	}, true
}

func structuredGoImpactEvidenceDisplayPathCandidates(absPath string, displayPath string, opts SearchOptions) []string {
	seen := make(map[string]struct{}, 2)
	candidates := make([]string, 0, 2)
	displayCandidates := []string{absPath}
	if structuredGoImpactRawEvidenceDisplayPathAllowed(opts) {
		displayCandidates = append(displayCandidates, displayPath)
	}
	for _, candidate := range displayCandidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		candidates = append(candidates, candidate)
	}
	return candidates
}

func structuredGoImpactEvidenceAbsolutePath(path string, opts SearchOptions) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}

	return absoluteAffectedFilePathWithPreferredBases(path, structuredGoImpactEvidenceBaseCandidates(opts)...)
}

func structuredGoImpactEvidenceBaseCandidates(opts SearchOptions) []string {
	candidates := make([]string, 0, 3)
	if target := structuredImpactSearchTargetPath(opts); target != "" {
		candidates = append(candidates, target)
	}
	if root := structuredImpactWorkspaceRoot(opts); root != "" && root != "." {
		candidates = append(candidates, root)
	}
	if cwd := invocationCWDOrGetwd(opts); cwd != "" {
		candidates = append(candidates, cwd)
	}
	return candidates
}
