package filequery

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
)

// SyntaxKind は direct query entry の構文分類を表す。
type SyntaxKind string

const (
	// SyntaxNone は direct query として扱わない構文を表す。
	SyntaxNone SyntaxKind = "none"
	// SyntaxExplicitPath は明示的な path/range/direct directory 構文を表す。
	SyntaxExplicitPath SyntaxKind = "explicit_path"
	// SyntaxPathCandidate は file-like な相対 path 候補を表す。
	SyntaxPathCandidate SyntaxKind = "path_candidate"
	// SyntaxBareExtFileCandidate は bare filename + extension の候補を表す。
	SyntaxBareExtFileCandidate SyntaxKind = "bare_ext_file_candidate"
	// SyntaxBareNamedFileCandidate は Makefile など特殊 bare filename 候補を表す。
	SyntaxBareNamedFileCandidate SyntaxKind = "bare_named_file_candidate"
)

// Input は direct query の entry 群を表す。
type Input struct {
	Entries []Entry
}

// Entry は direct query entry の構文情報を表す。
type Entry struct {
	RawEntry         string
	RawPath          string
	CleanedPath      string
	StartLine        int
	EndLine          int
	ExplicitRelative bool
	Syntax           SyntaxKind
}

// SplitEntries は comma 区切り direct query を entry 文字列へ分割する。
func SplitEntries(query string) []string {
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

// ParseInput は query 全体を direct query input として parse する。
func ParseInput(query string) (Input, bool) {
	rawEntries := SplitEntries(query)
	if len(rawEntries) == 0 {
		return Input{}, false
	}

	entries := make([]Entry, 0, len(rawEntries))
	for _, rawEntry := range rawEntries {
		entry, ok := ParseEntry(rawEntry)
		if !ok {
			return Input{}, false
		}
		entries = append(entries, entry)
	}

	return Input{Entries: entries}, true
}

// ParseEntry は direct query entry を path/range/syntax に分解する。
func ParseEntry(entry string) (Entry, bool) {
	rawEntry := strings.TrimSpace(entry)
	if rawEntry == "" {
		return Entry{}, false
	}

	rawPath, startLine, endLine := ParsePath(rawEntry)
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return Entry{}, false
	}

	input := Entry{
		RawEntry:         rawEntry,
		RawPath:          rawPath,
		CleanedPath:      filepath.Clean(rawPath),
		StartLine:        startLine,
		EndLine:          endLine,
		ExplicitRelative: HasExplicitRelativePrefix(rawPath),
	}
	input.Syntax = ClassifyEntrySyntax(input)
	return input, true
}

// ParsePath は path:line-range 形式を path と行範囲に分解する。
func ParsePath(entry string) (string, int, int) {
	lastColon := strings.LastIndex(entry, ":")
	if lastColon < 0 {
		return entry, 0, 0
	}

	suffix := entry[lastColon+1:]
	path := entry[:lastColon]

	if dashIdx := strings.Index(suffix, "-"); dashIdx >= 0 {
		startStr := suffix[:dashIdx]
		endStr := suffix[dashIdx+1:]
		start, err1 := strconv.Atoi(startStr)
		end, err2 := strconv.Atoi(endStr)
		if err1 == nil && err2 == nil && start > 0 && end > 0 {
			return path, start, end
		}
		return entry, 0, 0
	}

	start, err := strconv.Atoi(suffix)
	if err == nil && start > 0 {
		return path, start, 0
	}

	return entry, 0, 0
}

// LooksLikeExplicitQuery は query が明示的 direct path だけで構成されているか返す。
func LooksLikeExplicitQuery(query string) bool {
	input, ok := ParseInput(query)
	if !ok {
		return false
	}
	return InputHasOnlyExplicitPathSyntax(input)
}

// HasExplicitRelativePrefix は path が ./ や ../ で始まる明示相対 path か返す。
func HasExplicitRelativePrefix(path string) bool {
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

// ClassifyEntrySyntax は entry の direct query 構文種別を返す。
func ClassifyEntrySyntax(entry Entry) SyntaxKind {
	switch {
	case entryLooksLikeNaturalLanguageSearchIntent(entry):
		return SyntaxNone
	case LooksLikeExplicitDirectPath(entry):
		return SyntaxExplicitPath
	case LooksLikeDirectPathCandidate(entry):
		return SyntaxPathCandidate
	case LooksLikeBareExtFileCandidate(entry.RawPath):
		return SyntaxBareExtFileCandidate
	case LooksLikeBareNamedFileCandidate(entry.RawPath):
		return SyntaxBareNamedFileCandidate
	default:
		return SyntaxNone
	}
}

// LooksLikeExplicitDirectPath は entry が明示的 direct path か返す。
func LooksLikeExplicitDirectPath(entry Entry) bool {
	if entry.CleanedPath == "" {
		return false
	}
	if filepath.IsAbs(entry.CleanedPath) {
		return true
	}
	if HasWindowsPathPrefix(entry.RawPath) {
		return true
	}
	if entry.StartLine > 0 || entry.EndLine > 0 {
		return true
	}
	if HasExplicitDirectoryMarker(entry.RawPath) {
		return true
	}
	if HasExplicitRelativePrefix(entry.RawPath) {
		return true
	}
	return false
}

// LooksLikeDirectPathCandidate は entry が暗黙 direct path 候補か返す。
func LooksLikeDirectPathCandidate(entry Entry) bool {
	if entry.CleanedPath == "" {
		return false
	}
	if filepath.IsAbs(entry.CleanedPath) || HasWindowsPathPrefix(entry.RawPath) || entry.ExplicitRelative {
		return false
	}
	if entry.StartLine > 0 || entry.EndLine > 0 {
		return false
	}
	if strings.ContainsAny(entry.RawPath, " \t\r\n") {
		return false
	}
	if !strings.ContainsAny(entry.RawPath, `/\`) {
		return false
	}
	return PathCandidateHasFileLikeIntent(entry)
}

// LooksLikeBareExtFileCandidate は bare filename が拡張子付き file 候補か返す。
func LooksLikeBareExtFileCandidate(rawPath string) bool {
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
	if HasExplicitRelativePrefix(rawPath) || HasWindowsPathPrefix(rawPath) {
		return false
	}
	if rawPath == "." || rawPath == ".." {
		return false
	}

	ext := strings.TrimPrefix(filepath.Ext(rawPath), ".")
	if ext == "" {
		return false
	}
	return filefilter.SupportsBareExtension(ext)
}

// LooksLikeBareNamedFileCandidate は bare filename が特殊 file 候補か返す。
func LooksLikeBareNamedFileCandidate(rawPath string) bool {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" || strings.ContainsAny(rawPath, " \t\r\n") {
		return false
	}
	if strings.ContainsAny(rawPath, `/\`) {
		return false
	}
	if HasExplicitRelativePrefix(rawPath) || HasWindowsPathPrefix(rawPath) {
		return false
	}
	if rawPath == "." || rawPath == ".." {
		return false
	}
	_, ok := SpecialBareFileNames[rawPath]
	return ok
}

// InputHasOnlyExplicitPathSyntax は全 entry が明示 direct path 構文か返す。
func InputHasOnlyExplicitPathSyntax(input Input) bool {
	return InputHasOnlyEntrySyntax(input, SyntaxExplicitPath)
}

// InputHasOnlyPathCandidateSyntax は全 entry が path candidate 構文か返す。
func InputHasOnlyPathCandidateSyntax(input Input) bool {
	return InputHasOnlyEntrySyntax(input, SyntaxPathCandidate)
}

// InputHasOnlyStrongDirectIntent は全 entry が強い direct read 意図を持つか返す。
func InputHasOnlyStrongDirectIntent(input Input) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		if !EntryHasStrongDirectIntent(entry) {
			return false
		}
	}
	return true
}

// InputHasStrictScopedDirectIntent は scoped direct route で失敗を error にすべき意図か返す。
func InputHasStrictScopedDirectIntent(input Input) bool {
	if len(input.Entries) == 0 {
		return false
	}
	if len(input.Entries) == 1 {
		return EntryHasStrictScopedDirectIntent(input.Entries[0])
	}
	return InputHasOnlyScopedExactBatchIntent(input)
}

// InputContainsPathCandidateSyntax は path candidate 構文を含むか返す。
func InputContainsPathCandidateSyntax(input Input) bool {
	for _, entry := range input.Entries {
		if entry.Syntax == SyntaxPathCandidate {
			return true
		}
	}
	return false
}

// InputHasOnlyNamedBareFileCandidates は全 entry が特殊 bare filename 候補か返す。
func InputHasOnlyNamedBareFileCandidates(input Input) bool {
	return InputHasOnlyEntrySyntax(input, SyntaxBareNamedFileCandidate)
}

// InputHasOnlyCandidateDirectSyntax は全 entry が暗黙 direct 候補として扱えるか返す。
func InputHasOnlyCandidateDirectSyntax(input Input, allowNamedBareFiles bool) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		switch entry.Syntax {
		case SyntaxPathCandidate, SyntaxBareExtFileCandidate:
		case SyntaxBareNamedFileCandidate:
			if !allowNamedBareFiles {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// InputHasOnlyDirectReadCandidates は全 entry が direct read 候補として扱えるか返す。
func InputHasOnlyDirectReadCandidates(input Input, allowNamedBareFiles bool) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		switch entry.Syntax {
		case SyntaxExplicitPath, SyntaxBareExtFileCandidate:
		case SyntaxBareNamedFileCandidate:
			if !allowNamedBareFiles {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// InputHasOnlyEntrySyntax は全 entry が指定 syntax か返す。
func InputHasOnlyEntrySyntax(input Input, want SyntaxKind) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		if entry.Syntax != want {
			return false
		}
	}
	return true
}

// EntryHasStrongDirectIntent は entry が存在しなければ error にすべき direct 意図か返す。
func EntryHasStrongDirectIntent(entry Entry) bool {
	switch entry.Syntax {
	case SyntaxExplicitPath, SyntaxBareExtFileCandidate, SyntaxBareNamedFileCandidate:
		return true
	case SyntaxPathCandidate:
		return PathCandidateHasExactFileIntent(entry)
	default:
		return false
	}
}

// EntryHasStrictScopedDirectIntent は scoped lookup で entry の失敗を error にすべきか返す。
func EntryHasStrictScopedDirectIntent(entry Entry) bool {
	switch entry.Syntax {
	case SyntaxExplicitPath:
		return true
	case SyntaxBareExtFileCandidate, SyntaxBareNamedFileCandidate:
		return true
	case SyntaxPathCandidate:
		return PathCandidateHasExactFileIntent(entry)
	default:
		return false
	}
}

// InputHasOnlyScopedExactBatchIntent は scoped exact batch として扱える input か返す。
func InputHasOnlyScopedExactBatchIntent(input Input) bool {
	if len(input.Entries) == 0 {
		return false
	}
	for _, entry := range input.Entries {
		if !EntryHasScopedExactBatchIntent(entry) {
			return false
		}
	}
	return true
}

// EntryHasScopedExactBatchIntent は scoped exact batch の entry として扱えるか返す。
func EntryHasScopedExactBatchIntent(entry Entry) bool {
	switch entry.Syntax {
	case SyntaxBareExtFileCandidate, SyntaxBareNamedFileCandidate:
		return true
	case SyntaxPathCandidate:
		return PathCandidateHasExactFileIntent(entry)
	default:
		return false
	}
}

// PathCandidateHasExactFileIntent は path candidate が exact file 意図を持つか返す。
func PathCandidateHasExactFileIntent(entry Entry) bool {
	return PathCandidateHasFileLikeIntent(entry)
}

// PathCandidateHasFileLikeIntent は path candidate の basename が file-like か返す。
func PathCandidateHasFileLikeIntent(entry Entry) bool {
	baseName := strings.TrimSpace(filepath.Base(entry.CleanedPath))
	if baseName == "" || baseName == "." || baseName == ".." {
		return false
	}
	return HasBareFileExtension(baseName) || LooksLikeBareNamedFileCandidate(baseName)
}

// HasBareFileExtension は bare filename が拡張子を持つか返す。
func HasBareFileExtension(rawPath string) bool {
	rawPath = strings.TrimSpace(rawPath)
	if rawPath == "" {
		return false
	}
	return strings.TrimPrefix(filepath.Ext(rawPath), ".") != ""
}

// HasWindowsPathPrefix は Windows absolute path 風 prefix を持つか返す。
func HasWindowsPathPrefix(path string) bool {
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

// HasExplicitDirectoryMarker は path が末尾 slash による directory 指定か返す。
func HasExplicitDirectoryMarker(path string) bool {
	path = strings.TrimSpace(path)
	return strings.HasSuffix(path, "/") || strings.HasSuffix(path, `\`)
}

// SpecialBareFileNames は拡張子なしでも file として扱う bare filename の集合。
var SpecialBareFileNames = map[string]struct{}{
	"Makefile":   {},
	"Dockerfile": {},
}
