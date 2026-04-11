package file

import (
	"path/filepath"
	"strings"

	searchtool "github.com/susugadx/xelyon-cli/internal/tools/search"
)

type directQuerySyntaxKind string

const (
	directQuerySyntaxNone                   directQuerySyntaxKind = "none"
	directQuerySyntaxExplicitPath           directQuerySyntaxKind = "explicit_path"
	directQuerySyntaxPathCandidate          directQuerySyntaxKind = "path_candidate"
	directQuerySyntaxBareExtFileCandidate   directQuerySyntaxKind = "bare_ext_file_candidate"
	directQuerySyntaxBareNamedFileCandidate directQuerySyntaxKind = "bare_named_file_candidate"
)

type directQueryInput struct {
	entries []directQueryEntryInput
}

func splitDirectQueryEntries(query string) []string {
	parts := strings.Split(strings.TrimSpace(query), ",")
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil
		}
		entries = append(entries, part)
	}
	return entries
}

func parseDirectQueryInput(query string) (directQueryInput, bool) {
	rawEntries := splitDirectQueryEntries(query)
	if len(rawEntries) == 0 {
		return directQueryInput{}, false
	}

	entries := make([]directQueryEntryInput, 0, len(rawEntries))
	for _, rawEntry := range rawEntries {
		entry, ok := parseDirectQueryEntryInput(rawEntry)
		if !ok {
			return directQueryInput{}, false
		}
		entries = append(entries, entry)
	}

	return directQueryInput{entries: entries}, true
}

func parseDirectQueryEntryInput(entry string) (directQueryEntryInput, bool) {
	rawEntry := strings.TrimSpace(entry)
	if rawEntry == "" {
		return directQueryEntryInput{}, false
	}

	rawPath, startLine, endLine := parsePath(rawEntry)
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return directQueryEntryInput{}, false
	}

	input := directQueryEntryInput{
		rawEntry:         rawEntry,
		rawPath:          rawPath,
		cleanedPath:      filepath.Clean(rawPath),
		startLine:        startLine,
		endLine:          endLine,
		explicitRelative: hasExplicitRelativePrefix(rawPath),
	}
	input.syntax = classifyDirectQueryEntrySyntax(input)
	return input, true
}

func looksLikeExplicitDirectQuery(query string) bool {
	input, ok := parseDirectQueryInput(query)
	if !ok {
		return false
	}
	return inputHasOnlyExplicitPathSyntax(input)
}

func hasExplicitRelativePrefix(path string) bool {
	path = strings.TrimSpace(path)
	switch path {
	case ".", "..":
		return true
	}
	return strings.HasPrefix(path, "./") ||
		strings.HasPrefix(path, "../") ||
		strings.HasPrefix(path, ".\\") ||
		strings.HasPrefix(path, "..\\")
}

func classifyDirectQueryEntrySyntax(entry directQueryEntryInput) directQuerySyntaxKind {
	switch {
	case looksLikeExplicitDirectPath(entry):
		return directQuerySyntaxExplicitPath
	case looksLikeDirectPathCandidate(entry):
		return directQuerySyntaxPathCandidate
	case looksLikeBareExtFileCandidate(entry.rawPath):
		return directQuerySyntaxBareExtFileCandidate
	case looksLikeBareNamedFileCandidate(entry.rawPath):
		return directQuerySyntaxBareNamedFileCandidate
	default:
		return directQuerySyntaxNone
	}
}

func looksLikeExplicitDirectPath(entry directQueryEntryInput) bool {
	if entry.cleanedPath == "" {
		return false
	}
	if filepath.IsAbs(entry.cleanedPath) {
		return true
	}
	if hasWindowsPathPrefix(entry.rawPath) {
		return true
	}
	if entry.startLine > 0 || entry.endLine > 0 {
		return true
	}
	if hasExplicitDirectoryMarker(entry.rawPath) {
		return true
	}
	if hasExplicitRelativePrefix(entry.rawPath) {
		return true
	}
	return false
}

func looksLikeDirectPathCandidate(entry directQueryEntryInput) bool {
	if entry.cleanedPath == "" {
		return false
	}
	if filepath.IsAbs(entry.cleanedPath) || hasWindowsPathPrefix(entry.rawPath) || entry.explicitRelative {
		return false
	}
	if entry.startLine > 0 || entry.endLine > 0 {
		return false
	}
	if strings.ContainsAny(entry.rawPath, " \t\r\n") {
		return false
	}
	if !strings.ContainsAny(entry.rawPath, `/\`) {
		return false
	}
	return pathCandidateHasFileLikeIntent(entry)
}

func looksLikeBareExtFileCandidate(rawPath string) bool {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return false
	}
	if strings.ContainsAny(rawPath, " \t\r\n") {
		return false
	}
	if strings.ContainsAny(rawPath, `/\`) {
		return false
	}
	if hasExplicitRelativePrefix(rawPath) || hasWindowsPathPrefix(rawPath) {
		return false
	}
	if rawPath == "." || rawPath == ".." {
		return false
	}

	ext := strings.TrimPrefix(filepath.Ext(rawPath), ".")
	if ext == "" {
		return false
	}
	return searchtool.SupportsBareFileExtension(ext)
}

func looksLikeBareNamedFileCandidate(rawPath string) bool {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" || strings.ContainsAny(rawPath, " \t\r\n") {
		return false
	}
	if strings.ContainsAny(rawPath, `/\`) {
		return false
	}
	if hasExplicitRelativePrefix(rawPath) || hasWindowsPathPrefix(rawPath) {
		return false
	}
	if rawPath == "." || rawPath == ".." {
		return false
	}
	_, ok := directQuerySpecialBareFileNames[rawPath]
	return ok
}

func inputHasOnlyExplicitPathSyntax(input directQueryInput) bool {
	return inputHasOnlyEntrySyntax(input, directQuerySyntaxExplicitPath)
}

func inputHasOnlyPathCandidateSyntax(input directQueryInput) bool {
	return inputHasOnlyEntrySyntax(input, directQuerySyntaxPathCandidate)
}

func inputHasOnlyStrongDirectIntent(input directQueryInput) bool {
	if len(input.entries) == 0 {
		return false
	}
	for _, entry := range input.entries {
		if !entryHasStrongDirectIntent(entry) {
			return false
		}
	}
	return true
}

func inputHasStrictScopedDirectIntent(input directQueryInput) bool {
	if len(input.entries) == 0 {
		return false
	}
	if len(input.entries) == 1 {
		return entryHasStrictScopedDirectIntent(input.entries[0])
	}
	return inputHasOnlyScopedExactBatchIntent(input)
}

func inputContainsPathCandidateSyntax(input directQueryInput) bool {
	for _, entry := range input.entries {
		if entry.syntax == directQuerySyntaxPathCandidate {
			return true
		}
	}
	return false
}

func inputHasOnlyNamedBareFileCandidates(input directQueryInput) bool {
	return inputHasOnlyEntrySyntax(input, directQuerySyntaxBareNamedFileCandidate)
}

func inputHasOnlyCandidateDirectSyntax(input directQueryInput, allowNamedBareFiles bool) bool {
	if len(input.entries) == 0 {
		return false
	}
	for _, entry := range input.entries {
		switch entry.syntax {
		case directQuerySyntaxPathCandidate, directQuerySyntaxBareExtFileCandidate:
		case directQuerySyntaxBareNamedFileCandidate:
			if !allowNamedBareFiles {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func inputHasOnlyDirectReadCandidates(input directQueryInput, allowNamedBareFiles bool) bool {
	if len(input.entries) == 0 {
		return false
	}
	for _, entry := range input.entries {
		switch entry.syntax {
		case directQuerySyntaxExplicitPath, directQuerySyntaxBareExtFileCandidate:
		case directQuerySyntaxBareNamedFileCandidate:
			if !allowNamedBareFiles {
				return false
			}
		default:
			return false
		}
	}
	return true
}

func inputHasOnlyEntrySyntax(input directQueryInput, want directQuerySyntaxKind) bool {
	if len(input.entries) == 0 {
		return false
	}
	for _, entry := range input.entries {
		if entry.syntax != want {
			return false
		}
	}
	return true
}

func entryHasStrongDirectIntent(entry directQueryEntryInput) bool {
	switch entry.syntax {
	case directQuerySyntaxExplicitPath, directQuerySyntaxBareExtFileCandidate, directQuerySyntaxBareNamedFileCandidate:
		return true
	case directQuerySyntaxPathCandidate:
		return pathCandidateHasExactFileIntent(entry)
	default:
		return false
	}
}

func entryHasStrictScopedDirectIntent(entry directQueryEntryInput) bool {
	switch entry.syntax {
	case directQuerySyntaxExplicitPath:
		return true
	case directQuerySyntaxBareExtFileCandidate, directQuerySyntaxBareNamedFileCandidate:
		return true
	case directQuerySyntaxPathCandidate:
		return pathCandidateHasExactFileIntent(entry)
	default:
		return false
	}
}

func inputHasOnlyScopedExactBatchIntent(input directQueryInput) bool {
	if len(input.entries) == 0 {
		return false
	}
	for _, entry := range input.entries {
		if !entryHasScopedExactBatchIntent(entry) {
			return false
		}
	}
	return true
}

func entryHasScopedExactBatchIntent(entry directQueryEntryInput) bool {
	switch entry.syntax {
	case directQuerySyntaxBareExtFileCandidate, directQuerySyntaxBareNamedFileCandidate:
		return true
	case directQuerySyntaxPathCandidate:
		return pathCandidateHasExactFileIntent(entry)
	default:
		return false
	}
}

func pathCandidateHasExactFileIntent(entry directQueryEntryInput) bool {
	return pathCandidateHasFileLikeIntent(entry)
}

func pathCandidateHasFileLikeIntent(entry directQueryEntryInput) bool {
	baseName := strings.TrimSpace(filepath.Base(entry.cleanedPath))
	if baseName == "" || baseName == "." || baseName == ".." {
		return false
	}
	return hasBareFileExtension(baseName) || looksLikeBareNamedFileCandidate(baseName)
}

func hasBareFileExtension(rawPath string) bool {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return false
	}
	return strings.TrimPrefix(filepath.Ext(rawPath), ".") != ""
}

func hasWindowsPathPrefix(path string) bool {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, `\\`) || strings.HasPrefix(path, `//`) {
		return true
	}
	if len(path) < 2 || path[1] != ':' {
		return false
	}
	drive := path[0]
	return (drive >= 'a' && drive <= 'z') || (drive >= 'A' && drive <= 'Z')
}

func hasExplicitDirectoryMarker(path string) bool {
	path = strings.TrimSpace(path)
	return strings.HasSuffix(path, "/") || strings.HasSuffix(path, `\`)
}

var directQuerySpecialBareFileNames = map[string]struct{}{
	"Makefile":   {},
	"Dockerfile": {},
}
