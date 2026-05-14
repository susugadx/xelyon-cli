package search

import (
	"os"
	"sort"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

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
