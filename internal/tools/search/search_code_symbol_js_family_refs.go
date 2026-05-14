package search

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/susugadx/xelyon-cli/internal/jsast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

const jsFamilyLSPReferenceTimeout = 5 * time.Second
const jsFamilyBundleLSPSource = "TypeScript/JavaScript LSP"

type jsFamilyReferenceResult struct {
	refs           []genericSymbolRef
	resolvedViaLSP bool
}

type jsFamilyReferenceOptions struct {
	lsp      jsFamilyLSPReferenceOptions
	nameOnly SearchOptions
}

type jsFamilyLSPReferenceOptions struct {
	request  SearchOptions
	filter   SearchOptions
	location jsFamilyLSPLocationOptions
}

type jsFamilyLSPLocationOptions struct {
	adapterBase string
	displayRoot string
}

func newJSFamilyReferenceOptions(opts SearchOptions) jsFamilyReferenceOptions {
	return jsFamilyReferenceOptions{
		lsp:      newJSFamilyLSPReferenceOptions(opts, opts, opts),
		nameOnly: opts,
	}
}

func newJSFamilyLSPReferenceOptions(request SearchOptions, filter SearchOptions, locationBase SearchOptions) jsFamilyLSPReferenceOptions {
	return jsFamilyLSPReferenceOptions{
		request:  request,
		filter:   filter,
		location: newJSFamilyLSPLocationOptions(locationBase),
	}
}

func newJSFamilyLSPLocationOptions(opts SearchOptions) jsFamilyLSPLocationOptions {
	return jsFamilyLSPLocationOptions{
		adapterBase: invocationCWDOrGetwd(opts),
		displayRoot: affectedFileBasePath(opts, affectedFileSourceText),
	}
}

func setJSFamilyBundleLSPDiagnostics(bundle *SymbolBundle, resolved bool) {
	if bundle == nil {
		return
	}
	bundle.Diagnostics.ResolvedViaLSP = resolved
	if resolved {
		bundle.Diagnostics.LSPSource = jsFamilyBundleLSPSource
	}
}

func findJSFamilyReferencesWithSemantic(symbol string, def genericSymbolDef, opts jsFamilyReferenceOptions) jsFamilyReferenceResult {
	if opts.lsp.request.LSPClient != nil && def.Character > 0 {
		refs, err := findJSFamilyReferencesWithLSP(symbol, def, opts.lsp)
		if err == nil && len(refs) > 0 {
			return jsFamilyReferenceResult{refs: refs, resolvedViaLSP: true}
		}
	}
	return jsFamilyReferenceResult{refs: findJSFamilyReferencesWithAST(symbol, opts.nameOnly)}
}

func findJSFamilyReferencesWithLSP(symbol string, def genericSymbolDef, opts jsFamilyLSPReferenceOptions) ([]genericSymbolRef, error) {
	defPath := absoluteAffectedFilePath(def.File, opts.request, affectedFileSourceText)
	if defPath == "" {
		defPath = absoluteAffectedFilePathWithBase(def.File, invocationCWDOrGetwd(opts.request))
	}
	if defPath == "" {
		return nil, os.ErrNotExist
	}

	ctx, cancel := context.WithTimeout(context.Background(), jsFamilyLSPReferenceTimeout)
	defer cancel()
	locations, err := opts.request.LSPClient.FindReferences(ctx, defPath, def.Line, def.Character, false)
	if err != nil {
		return nil, err
	}

	refs := make([]genericSymbolRef, 0, len(locations))
	for _, loc := range locations {
		ref, ok := jsFamilyRefFromLSPLocation(symbol, loc, opts)
		if ok {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func jsFamilyRefFromLSPLocation(symbol string, loc navigation.LSPLocation, opts jsFamilyLSPReferenceOptions) (genericSymbolRef, bool) {
	displayPath, absPath := jsFamilyLSPPaths(loc.File, opts.location)
	if displayPath == "" || absPath == "" || loc.Line <= 0 {
		return genericSymbolRef{}, false
	}
	if !jsFamilyLSPReferenceAllowed(absPath, displayPath, opts.filter) {
		return genericSymbolRef{}, false
	}

	ref := genericSymbolRef{
		File:    displayPath,
		Line:    loc.Line,
		Snippet: jsFamilyLineSnippet(absPath, loc.Line),
		IsTest:  repomap.IsTestFile(displayPath),
	}
	if parsed, ok := parseJSFamilyFileForSearch(absPath); ok {
		if info, err := jsast.ClassifyRangeWithParsed(parsed, loc.Line, loc.Character, loc.EndLine, loc.EndChar, symbol); err == nil && info != nil {
			ref.Class = info.Class
		}
		parsed.Close()
	}
	return ref, true
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

func jsFamilyLineSnippet(absPath string, line int) string {
	src, err := os.ReadFile(absPath)
	if err != nil || line <= 0 {
		return ""
	}
	lines := strings.Split(string(src), "\n")
	if line > len(lines) {
		return ""
	}
	return strings.TrimSpace(lines[line-1])
}
