package search

import (
	"path/filepath"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

func findJSFamilyReferencesWithAST(symbol string, opts SearchOptions) []genericSymbolRef {
	collector := newJSFamilyASTReferenceCollector(symbol, opts, maxGenericRefs)
	defer collector.Close()
	streamGenericSymbolMatches(symbol, opts, collector.Add)
	return collector.refs
}

func classifyJSFamilySymbolRefsFromAST(refs []genericSymbolRef) jsFamilySymbolRefs {
	var classified jsFamilySymbolRefs
	for _, ref := range refs {
		if !jsFamilyReferenceClassVisible(ref.Class) {
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
		default:
			classified.others = append(classified.others, ref)
		}
	}
	return classified
}

type jsFamilyASTReferenceCollector struct {
	symbol string
	opts   SearchOptions
	limit  int
	refs   []genericSymbolRef
	files  map[string]*jsast.ParsedFile
}

func newJSFamilyASTReferenceCollector(symbol string, opts SearchOptions, limit int) *jsFamilyASTReferenceCollector {
	return &jsFamilyASTReferenceCollector{
		symbol: symbol,
		opts:   opts,
		limit:  limit,
		refs:   make([]genericSymbolRef, 0, limit),
		files:  make(map[string]*jsast.ParsedFile),
	}
}

func (c *jsFamilyASTReferenceCollector) Add(match genericSymbolMatch) bool {
	ref := genericSymbolRefFromMatch(match)
	c.classify(&ref)
	if !jsFamilyReferenceClassVisible(ref.Class) {
		return true
	}

	c.refs = append(c.refs, ref)
	return c.limit <= 0 || len(c.refs) < c.limit
}

func (c *jsFamilyASTReferenceCollector) classify(ref *genericSymbolRef) {
	abs := absoluteAffectedFilePath(ref.File, c.opts, affectedFileSourceText)
	if abs == "" || !jsast.Supports(abs) {
		return
	}
	parsed := c.parsedFile(abs)
	if parsed == nil {
		return
	}
	info, err := jsast.ClassifyLineWithParsed(parsed, ref.Line, c.symbol)
	if err != nil || info == nil {
		return
	}
	ref.Class = info.Class
}

func (c *jsFamilyASTReferenceCollector) parsedFile(absPath string) *jsast.ParsedFile {
	key := filepath.Clean(absPath)
	if parsed, ok := c.files[key]; ok {
		return parsed
	}

	parsed, ok := parseJSFamilyFileForSearch(key)
	if !ok {
		c.files[key] = nil
		return nil
	}
	c.files[key] = parsed
	return parsed
}

func (c *jsFamilyASTReferenceCollector) Close() {
	for _, parsed := range c.files {
		if parsed != nil {
			parsed.Close()
		}
	}
}

func jsFamilyReferenceClassVisible(class codeast.MatchClass) bool {
	switch class {
	case codeast.ClassDef, codeast.ClassString, codeast.ClassComment, jsast.ClassIgnored:
		return false
	default:
		return true
	}
}
