package search

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

const jsFamilyLSPReferenceTimeout = 5 * time.Second

type jsFamilyReferenceResult struct {
	refs           []genericSymbolRef
	resolvedViaLSP bool
}

func findJSFamilyDefinitionsWithAST(symbol string, opts SearchOptions) []genericSymbolDef {
	matches := findGenericSymbolMatches(symbol, opts, 0)
	files := jsFamilyMatchedFiles(matches, opts)
	defs := make([]genericSymbolDef, 0)
	for _, file := range files {
		parsed, ok := parseJSFamilyFileForSearch(file.abs)
		if !ok {
			continue
		}
		symbols := jsast.ExtractSymbolsWithParsed(parsed)
		parsed.Close()
		for _, astSymbol := range symbols {
			if astSymbol.Name != symbol {
				continue
			}
			defs = append(defs, genericSymbolDef{
				Name:      astSymbol.Name,
				Kind:      astSymbol.Kind,
				File:      file.display,
				Line:      astSymbol.Line,
				Character: astSymbol.Character,
				Signature: astSymbol.Signature,
				Exported:  astSymbol.Exported,
			})
		}
	}
	return defs
}

func findJSFamilyReferencesWithSemantic(symbol string, def genericSymbolDef, opts SearchOptions) jsFamilyReferenceResult {
	if opts.LSPClient != nil && def.Character > 0 {
		refs, err := findJSFamilyReferencesWithLSP(symbol, def, opts)
		if err == nil && len(refs) > 0 {
			return jsFamilyReferenceResult{refs: refs, resolvedViaLSP: true}
		}
	}
	return jsFamilyReferenceResult{refs: findJSFamilyReferencesWithAST(symbol, opts)}
}

func findJSFamilyReferencesWithLSP(symbol string, def genericSymbolDef, opts SearchOptions) ([]genericSymbolRef, error) {
	defPath := absoluteAffectedFilePath(def.File, opts, affectedFileSourceText)
	if defPath == "" {
		defPath = absoluteAffectedFilePathWithBase(def.File, invocationCWDOrGetwd(opts))
	}
	if defPath == "" {
		return nil, os.ErrNotExist
	}

	ctx, cancel := context.WithTimeout(context.Background(), jsFamilyLSPReferenceTimeout)
	defer cancel()
	locations, err := opts.LSPClient.FindReferences(ctx, defPath, def.Line, def.Character, false)
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

func jsFamilyRefFromLSPLocation(symbol string, loc navigation.LSPLocation, opts SearchOptions) (genericSymbolRef, bool) {
	displayPath, absPath := jsFamilyLSPPaths(loc.File, opts)
	if displayPath == "" || absPath == "" || loc.Line <= 0 {
		return genericSymbolRef{}, false
	}
	if !jsFamilyLSPReferenceAllowed(absPath, displayPath, opts) {
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

func jsFamilyLSPPaths(file string, opts SearchOptions) (string, string) {
	file = strings.TrimSpace(file)
	if file == "" {
		return "", ""
	}
	root := invocationCWDOrGetwd(opts)
	cleanFile := filepath.Clean(file)
	absPath := cleanFile
	displayPath := cleanFile
	if filepath.IsAbs(cleanFile) {
		if root != "" {
			if rel, err := filepath.Rel(root, cleanFile); err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				displayPath = rel
			}
		}
	} else {
		absPath = absoluteAffectedFilePathWithBase(cleanFile, root)
		if absPath == "" {
			absPath = absoluteAffectedFilePath(cleanFile, opts, affectedFileSourceText)
		}
	}
	if absPath == "" {
		return "", ""
	}
	return filepath.ToSlash(filepath.Clean(displayPath)), absPath
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

func findJSFamilyReferencesWithAST(symbol string, opts SearchOptions) []genericSymbolRef {
	refs := findGenericReferences(symbol, opts)
	grouped := make(map[string][]int)
	for i := range refs {
		abs := absoluteAffectedFilePath(refs[i].File, opts, affectedFileSourceText)
		if abs == "" || !jsast.Supports(abs) {
			continue
		}
		grouped[abs] = append(grouped[abs], i)
	}

	for abs, indexes := range grouped {
		parsed, ok := parseJSFamilyFileForSearch(abs)
		if !ok {
			continue
		}
		for _, idx := range indexes {
			info, err := jsast.ClassifyLineWithParsed(parsed, refs[idx].Line, symbol)
			if err != nil || info == nil {
				continue
			}
			refs[idx].Class = info.Class
		}
		parsed.Close()
	}
	return refs
}

func classifyJSFamilySymbolRefsFromAST(refs []genericSymbolRef) jsFamilySymbolRefs {
	var classified jsFamilySymbolRefs
	for _, ref := range refs {
		switch ref.Class {
		case codeast.ClassString, codeast.ClassComment, jsast.ClassIgnored:
			continue
		}
		if ref.IsTest {
			classified.tests = append(classified.tests, ref)
			continue
		}
		switch ref.Class {
		case codeast.ClassImport:
			classified.imports = append(classified.imports, ref)
		case jsast.ClassExport:
			classified.imports = append(classified.imports, ref)
		case codeast.ClassCall:
			classified.callers = append(classified.callers, ref)
		case jsast.ClassTypeRef:
			classified.typeRefs = append(classified.typeRefs, ref)
		case codeast.ClassDef:
			continue
		default:
			classified.others = append(classified.others, ref)
		}
	}
	return classified
}

type jsFamilyMatchedFile struct {
	display string
	abs     string
}

func jsFamilyMatchedFiles(matches []genericSymbolMatch, opts SearchOptions) []jsFamilyMatchedFile {
	seen := make(map[string]struct{})
	files := make([]jsFamilyMatchedFile, 0)
	for _, match := range matches {
		abs := absoluteAffectedFilePath(match.File, opts, affectedFileSourceText)
		if abs == "" || !jsast.Supports(abs) {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		files = append(files, jsFamilyMatchedFile{display: match.File, abs: abs})
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].display < files[j].display
	})
	return files
}

func parseJSFamilyFileForSearch(absPath string) (*jsast.ParsedFile, bool) {
	src, err := os.ReadFile(absPath)
	if err != nil {
		return nil, false
	}
	parsed, err := jsast.ParseBytes(absPath, src)
	if err != nil {
		return nil, false
	}
	return parsed, true
}
