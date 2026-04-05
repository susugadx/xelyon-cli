package search

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
)

type goMethodTestProbeContext struct {
	probe                string
	symbol               navigation.SymbolCandidate
	opts                 SearchOptions
	packageDir           string
	targetPackageDir     string
	receiver             string
	requireReceiver      bool
	alternativeReceivers map[string]struct{}
	localTypeNames       map[string]struct{}
	dependencies         *structuredGoImpactProbeDeps
}

func newGoMethodTestProbeContext(probe string, symbol navigation.SymbolCandidate, opts SearchOptions, packageDir string, entries []os.DirEntry, deps *structuredGoImpactProbeDeps) (goMethodTestProbeContext, bool) {
	receiver := structuredGoImpactProbeReceiver(symbol)
	if receiver == "" {
		return goMethodTestProbeContext{}, false
	}

	alternativeReceivers := methodProbeAlternativeReceivers(entries, packageDir, symbol, opts, deps)
	localTypeNames := methodProbeLocalTypeNames(entries, packageDir, symbol, opts, deps)
	return goMethodTestProbeContext{
		probe:                probe,
		symbol:               symbol,
		opts:                 opts,
		packageDir:           packageDir,
		targetPackageDir:     packageDir,
		receiver:             receiver,
		requireReceiver:      len(alternativeReceivers) > 0,
		alternativeReceivers: alternativeReceivers,
		localTypeNames:       localTypeNames,
		dependencies:         deps,
	}, true
}

func (ctx goMethodTestProbeContext) collectLocalTests(entries []os.DirEntry, appendTest func(navigation.TestRef)) {
	for _, name := range ctx.collectCandidateTestFileNames(entries) {
		ctx.appendLocalTestsFromFile(filepath.Join(ctx.packageDir, name), appendTest)
	}
}

func (ctx goMethodTestProbeContext) collectCandidateTestFileNames(entries []os.DirEntry) []string {
	var names []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		if !shouldIncludeStructuredGoImpactProbeFile(filepath.Join(ctx.packageDir, entry.Name()), ctx.symbol.RootPath, ctx.opts) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}

func (ctx goMethodTestProbeContext) appendLocalTestsFromFile(absPath string, appendTest func(navigation.TestRef)) {
	ctx.dependencies.add(absPath)
	src, err := os.ReadFile(absPath)
	if err != nil {
		return
	}
	if len(ast.ValidateSyntax(absPath, src)) > 0 {
		for _, test := range fallbackGoImpactTestsByNameProbeInFile(src, absPath, ctx.probe, ctx.symbol.RootPath, ctx.opts) {
			appendTest(test)
		}
		return
	}

	parsed, err := ast.ParseBytesForReuse(absPath, src)
	if err != nil {
		for _, test := range fallbackGoImpactTestsByNameProbeInFile(src, absPath, ctx.probe, ctx.symbol.RootPath, ctx.opts) {
			appendTest(test)
		}
		return
	}

	symbols, err := ast.ExtractSymbolsFromBytes(absPath, src)
	if err != nil {
		for _, test := range fallbackGoImpactTestsByNameProbeInFile(src, absPath, ctx.probe, ctx.symbol.RootPath, ctx.opts) {
			appendTest(test)
		}
		return
	}

	matchCtx := ctx.matchContext(absPath, src)
	for _, candidate := range symbols {
		if candidate.Kind != ast.SymbolFunction || !strings.HasPrefix(candidate.Name, "Test") || !strings.Contains(candidate.Name, ctx.probe) {
			continue
		}
		if !methodTestBodyMatchesSymbol(matchCtx, parsed, candidate, true) {
			continue
		}
		appendTest(navigation.TestRef{
			File: normalizeImpactProbeFile(absPath, ctx.symbol.RootPath, ctx.opts),
			Name: candidate.Name,
			Line: candidate.Line,
		})
	}
}

func methodProbeAlternativeReceivers(entries []os.DirEntry, packageDir string, symbol navigation.SymbolCandidate, opts SearchOptions, deps *structuredGoImpactProbeDeps) map[string]struct{} {
	targetReceiver := structuredGoImpactProbeReceiver(symbol)
	if targetReceiver == "" {
		return nil
	}

	alternatives := make(map[string]struct{})
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		absPath := filepath.Join(packageDir, entry.Name())
		if !shouldIncludeStructuredGoImpactProbeFile(absPath, symbol.RootPath, opts) {
			continue
		}
		deps.add(absPath)
		symbols, err := ast.ExtractSymbols(absPath)
		if err != nil {
			continue
		}
		for _, candidate := range symbols {
			if candidate.Kind != ast.SymbolMethod || candidate.Name != symbol.Name {
				continue
			}
			receiver := canonicalProbeReceiver(extractProbeMethodReceiver(candidate.Signature))
			if receiver != "" && receiver != targetReceiver {
				alternatives[receiver] = struct{}{}
			}
		}
	}
	return alternatives
}

func methodProbeLocalTypeNames(entries []os.DirEntry, packageDir string, symbol navigation.SymbolCandidate, opts SearchOptions, deps *structuredGoImpactProbeDeps) map[string]struct{} {
	localTypes := make(map[string]struct{})
	targetReceiver := structuredGoImpactProbeReceiver(symbol)
	if targetReceiver != "" {
		localTypes[targetReceiver] = struct{}{}
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") {
			continue
		}
		absPath := filepath.Join(packageDir, entry.Name())
		if !shouldIncludeStructuredGoImpactProbeFile(absPath, symbol.RootPath, opts) {
			continue
		}
		deps.add(absPath)
		symbols, err := ast.ExtractSymbols(absPath)
		if err != nil {
			continue
		}
		for _, candidate := range symbols {
			switch candidate.Kind {
			case ast.SymbolType, ast.SymbolStruct, ast.SymbolInterface:
				if name := strings.TrimSpace(candidate.Name); name != "" {
					localTypes[name] = struct{}{}
				}
			}
		}
	}
	return localTypes
}

func shouldIncludeStructuredGoImpactProbeFile(absPath, rootPath string, opts SearchOptions) bool {
	file := normalizeImpactProbeFile(absPath, rootPath, opts)
	if file == "" {
		return false
	}
	if !opts.IncludeHidden && structuredGoImpactProbePathHasHiddenSegment(file) {
		return false
	}
	if opts.ignoreMatcher != nil && opts.ignoreMatcher.Match(file, false) {
		return false
	}
	return matchesSearchFileFilter(file, opts)
}

func structuredGoImpactProbePathHasHiddenSegment(path string) bool {
	cleanPath := filepath.ToSlash(filepath.Clean(path))
	for _, segment := range strings.Split(cleanPath, "/") {
		if segment == "" || segment == "." || segment == ".." {
			continue
		}
		if strings.HasPrefix(segment, ".") {
			return true
		}
	}
	return false
}

func fallbackGoImpactTestsByNameProbeInFile(src []byte, absPath, probe, rootPath string, opts SearchOptions) []navigation.TestRef {
	file := normalizeImpactProbeFile(absPath, rootPath, opts)
	if file == "" {
		return nil
	}

	lines := strings.Split(string(src), "\n")
	tests := make([]navigation.TestRef, 0, 1)
	seen := make(map[string]struct{})
	for idx, line := range lines {
		name := extractGoImpactTestName(line, probe)
		if name == "" {
			continue
		}
		key := fmt.Sprintf("%s:%d:%s", file, idx+1, name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tests = append(tests, navigation.TestRef{
			File: file,
			Name: name,
			Line: idx + 1,
		})
	}
	return tests
}

func structuredGoImpactMethodProbePathIsFile(opts SearchOptions) bool {
	path := strings.TrimSpace(opts.Path)
	if path == "" {
		return false
	}
	absPath := absoluteAffectedFilePathWithBase(path, invocationCWDOrGetwd(opts))
	if absPath == "" {
		return false
	}
	info, err := os.Stat(absPath)
	return err == nil && !info.IsDir()
}

func structuredGoImpactMethodProbeDir(symbol navigation.SymbolCandidate, opts SearchOptions) string {
	if absPath := absoluteAffectedFilePathForSymbol(symbol.File, opts, symbol.RootPath); absPath != "" {
		return filepath.Dir(absPath)
	}
	if path := strings.TrimSpace(opts.Path); path != "" {
		if absPath := absoluteAffectedFilePathWithBase(path, invocationCWDOrGetwd(opts)); absPath != "" {
			if info, err := os.Stat(absPath); err == nil && info.IsDir() {
				return absPath
			}
			return filepath.Dir(absPath)
		}
	}
	return ""
}

func findGoImpactTestsByNameProbe(probe, rootPath string, opts SearchOptions, limit int) ([]navigation.TestRef, int) {
	probeOpts := opts
	probeOpts.Intent = ""
	probeOpts.Mode = string(SearchModeLiteral)
	probeOpts.IsRegex = false
	if strings.TrimSpace(probeOpts.FilePattern) == "" && strings.TrimSpace(probeOpts.FileType) == "" {
		probeOpts.FilePattern = "*_test.go"
	}

	output, useRipgrep, _, err := executeSearch(probe, probeOpts)
	if err != nil || strings.TrimSpace(output) == "" {
		return nil, 0
	}

	var results []SearchResult
	if useRipgrep {
		results = parseRipgrepJSON(output, 0)
	} else {
		results = parseGrepOutput(output, 0)
	}
	results = filterResultsByOptions(results, probeOpts)

	storeLimit := limit
	if storeLimit <= 0 || storeLimit > len(results) {
		storeLimit = len(results)
	}

	seen := make(map[string]struct{})
	tests := make([]navigation.TestRef, 0, storeLimit)
	total := 0
	for _, result := range results {
		file := normalizeImpactProbeFile(result.FilePath, rootPath, probeOpts)
		if file == "" {
			continue
		}
		for _, match := range result.Matches {
			if !match.IsMatch {
				continue
			}
			name := extractGoImpactTestName(match.Line, probe)
			if name == "" {
				continue
			}
			key := fmt.Sprintf("%s:%d:%s", file, match.LineNum, name)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			total++
			if limit <= 0 || len(tests) < limit {
				tests = append(tests, navigation.TestRef{
					File: file,
					Name: name,
					Line: match.LineNum,
				})
			}
		}
	}
	return tests, total
}

func normalizeImpactProbeFile(file, rootPath string, opts SearchOptions) string {
	absPath := absoluteAffectedFilePath(file, opts, affectedFileSourceText)
	if absPath == "" {
		return ""
	}

	rootPath = strings.TrimSpace(rootPath)
	if rootPath != "" {
		if rel, err := filepath.Rel(rootPath, absPath); err == nil {
			rel = filepath.Clean(rel)
			if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				return filepath.ToSlash(rel)
			}
		}
	}

	return filepath.ToSlash(filepath.Clean(file))
}

func extractGoImpactTestName(line, probe string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || !strings.Contains(trimmed, probe) || !strings.Contains(trimmed, "func ") {
		return ""
	}

	idx := strings.Index(trimmed, "func ")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(trimmed[idx+5:])
	if rest == "" {
		return ""
	}
	end := strings.Index(rest, "(")
	if end <= 0 {
		return ""
	}
	name := strings.TrimSpace(rest[:end])
	if name == "" || !strings.HasPrefix(name, "Test") {
		return ""
	}
	return name
}
