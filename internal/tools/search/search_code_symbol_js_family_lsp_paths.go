package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/jsast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type jsFamilyLSPReferenceCandidate struct {
	displayPath string
	absPath     string
	loc         navigation.LSPLocation
}

func jsFamilyLSPReferenceCandidateFromLocation(loc navigation.LSPLocation, opts jsFamilyLSPReferenceOptions) (jsFamilyLSPReferenceCandidate, bool) {
	displayPath, absPath := jsFamilyLSPPaths(loc.File, opts.location)
	if displayPath == "" || absPath == "" || loc.Line <= 0 {
		return jsFamilyLSPReferenceCandidate{}, false
	}
	if !jsFamilyLSPReferenceAllowed(absPath, displayPath, opts.filter) {
		return jsFamilyLSPReferenceCandidate{}, false
	}

	return jsFamilyLSPReferenceCandidate{
		displayPath: displayPath,
		absPath:     absPath,
		loc:         loc,
	}, true
}

func jsFamilyLSPReferenceAllowed(absPath string, displayPath string, opts SearchOptions) bool {
	if !jsast.Supports(absPath) {
		return false
	}
	return jsFamilySearchCandidateAllowed(absPath, displayPath, opts)
}

func jsFamilyLSPPaths(file string, opts jsFamilyLSPLocationOptions) (string, string) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", ""
	}
	cleanFile := filepath.Clean(file)
	var absPath string
	if filepath.IsAbs(cleanFile) {
		absPath = filepath.Clean(cleanFile)
	} else {
		absPath = absoluteAffectedFilePathWithPreferredBases(cleanFile, opts.adapterBase, opts.displayRoot)
	}
	if absPath == "" {
		return "", ""
	}
	displayPath := jsFamilyLSPDisplayPath(absPath, cleanFile, opts)
	return filepath.ToSlash(filepath.Clean(displayPath)), absPath
}

func jsFamilyLSPDisplayPath(absPath string, fallback string, opts jsFamilyLSPLocationOptions) string {
	if rel, ok := jsFamilyRelativePathInBase(absPath, opts.displayRoot); ok {
		return rel
	}
	if rel, ok := jsFamilyRelativePathInBase(absPath, opts.adapterBase); ok {
		return rel
	}
	if !filepath.IsAbs(fallback) {
		return fallback
	}
	return absPath
}

func jsFamilyRelativePathInBase(path string, base string) (string, bool) {
	base = normalizeAffectedFileBase(base)
	if base == "" {
		return "", false
	}
	cleanPath := filepath.Clean(path)
	rel, err := filepath.Rel(base, cleanPath)
	if err != nil {
		return "", false
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(filepath.Clean(rel)), true
}
