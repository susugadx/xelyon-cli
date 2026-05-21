package search

import (
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

func findJSFamilyReferencesWithAST(symbol string, def genericSymbolDef, opts SearchOptions) []genericSymbolRef {
	collector := newJSFamilyASTReferenceCollector(symbol, def, opts, maxGenericRefs)
	defer collector.Close()
	collector.CollectNameMatches()
	return collector.Result()
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
	symbol           string
	def              genericSymbolDef
	opts             SearchOptions
	limit            int
	refs             []genericSymbolRef
	aliasRefs        []genericSymbolRef
	aliasImports     []genericSymbolRef
	files            *jsFamilyASTParsedFileCache
	completedAliases map[jsFamilyASTAliasCompletionKey]struct{}
	aliasesExpanded  bool
}

func newJSFamilyASTReferenceCollector(symbol string, def genericSymbolDef, opts SearchOptions, limit int) *jsFamilyASTReferenceCollector {
	return &jsFamilyASTReferenceCollector{
		symbol: symbol,
		def:    def,
		opts:   opts,
		limit:  limit,
		refs:   make([]genericSymbolRef, 0, limit),
		files:  newJSFamilyASTParsedFileCache(),
	}
}

func (c *jsFamilyASTReferenceCollector) CollectNameMatches() {
	streamGenericSymbolMatches(c.symbol, c.opts, c.AddNameMatch)
}

func (c *jsFamilyASTReferenceCollector) Result() []genericSymbolRef {
	return c.refsWithJSXAliasUsages()
}

func (c *jsFamilyASTReferenceCollector) AddNameMatch(match genericSymbolMatch) bool {
	ref, ok := c.refFromNameMatch(match)
	if !ok {
		return true
	}
	continueStream := c.addRef(ref)
	if ref.Class != codeast.ClassImport {
		return continueStream
	}
	c.recordJSXAliasImport(ref)
	return continueStream
}

func (c *jsFamilyASTReferenceCollector) refFromNameMatch(match genericSymbolMatch) (genericSymbolRef, bool) {
	ref := genericSymbolRefFromMatch(match)
	c.classify(&ref)
	if !jsFamilyReferenceClassVisible(ref.Class) {
		return genericSymbolRef{}, false
	}
	return ref, true
}

func (c *jsFamilyASTReferenceCollector) addRef(ref genericSymbolRef) bool {
	c.refs = append(c.refs, ref)
	return c.limit <= 0 || len(c.refs) < c.limit
}

func (c *jsFamilyASTReferenceCollector) classify(ref *genericSymbolRef) {
	parsed := c.parsedFileForRefFile(ref.File)
	if parsed == nil {
		return
	}
	info, err := jsast.ClassifyLineWithParsed(parsed, ref.Line, c.symbol)
	if err != nil || info == nil {
		return
	}
	ref.Class = info.Class
}

func (c *jsFamilyASTReferenceCollector) parsedFileForRefFile(file string) *jsast.ParsedFile {
	abs := absoluteAffectedFilePath(file, c.opts, affectedFileSourceText)
	if abs == "" || !jsast.Supports(abs) {
		return nil
	}
	return c.files.Parsed(abs)
}

func (c *jsFamilyASTReferenceCollector) Close() {
	if c.files != nil {
		c.files.Close()
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
