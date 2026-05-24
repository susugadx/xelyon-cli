package search

import (
	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

func findJSFamilyReferencesWithAST(symbol string, def genericSymbolDef, opts SearchOptions) []genericSymbolRef {
	return findJSFamilyReferencesWithASTDetailed(symbol, def, opts).refs
}

type jsFamilyASTReferenceResult struct {
	refs           []genericSymbolRef
	rawMatchCount  int
	incomplete     bool
	truncated      bool
	budgetLimitHit bool
}

func findJSFamilyReferencesWithASTDetailed(symbol string, def genericSymbolDef, opts SearchOptions) jsFamilyASTReferenceResult {
	collector := newJSFamilyASTReferenceCollector(symbol, def, opts, maxGenericRefs)
	defer collector.Close()
	collector.CollectNameMatches()
	return collector.DetailedResult()
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
	symbol                  string
	def                     genericSymbolDef
	opts                    SearchOptions
	limit                   int
	refs                    []genericSymbolRef
	rawMatchCount           int
	aliasUsageCount         int
	incomplete              bool
	truncated               bool
	budgetLimitHit          bool
	importBindingRefs       []genericSymbolRef
	importBindingImports    []genericSymbolRef
	files                   *jsFamilyASTParsedFileCache
	completedImportBindings map[jsFamilyASTImportBindingCompletionKey]struct{}
	importBindingsExpanded  bool
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
	c.recordStreamResult(streamGenericSymbolMatches(c.symbol, c.opts, c.AddNameMatch))
	c.CollectDefaultImportSourceMatches()
}

func (c *jsFamilyASTReferenceCollector) CollectDefaultImportSourceMatches() {
	if !c.definitionAllowsDefaultImport() {
		return
	}
	c.recordStreamResult(streamGenericSymbolMatches("import", c.opts, c.AddDefaultImportSourceMatch))
	c.recordStreamResult(streamGenericSymbolMatches("require", c.opts, c.AddDefaultImportSourceMatch))
}

func (c *jsFamilyASTReferenceCollector) Result() []genericSymbolRef {
	return c.refsWithImportBindingUsages()
}

func (c *jsFamilyASTReferenceCollector) DetailedResult() jsFamilyASTReferenceResult {
	refs := c.Result()
	return jsFamilyASTReferenceResult{
		refs:           refs,
		rawMatchCount:  c.rawMatchCount + c.aliasUsageCount,
		incomplete:     c.incomplete,
		truncated:      c.truncated,
		budgetLimitHit: c.budgetLimitHit,
	}
}

func (c *jsFamilyASTReferenceCollector) recordStreamResult(result genericSymbolSearchResult) {
	if result.cancelRequested && !c.truncated {
		c.incomplete = true
	}
}

func (c *jsFamilyASTReferenceCollector) AddNameMatch(match genericSymbolMatch) bool {
	c.rawMatchCount++
	ref, ok := c.refFromNameMatch(match)
	if !ok {
		ref, ok = c.importBindingRefFromSourceMatch(match)
		if !ok {
			return true
		}
	}
	continueStream := c.addRef(ref)
	if ref.Class != codeast.ClassImport {
		return continueStream
	}
	c.recordImportBindingImport(ref)
	return continueStream
}

func (c *jsFamilyASTReferenceCollector) AddDefaultImportSourceMatch(match genericSymbolMatch) bool {
	c.rawMatchCount++
	ref, ok := c.defaultImportBindingRefFromSourceMatch(match)
	if !ok {
		return true
	}
	continueStream := c.addRef(ref)
	c.recordImportBindingImport(ref)
	return continueStream
}

func (c *jsFamilyASTReferenceCollector) refFromNameMatch(match genericSymbolMatch) (genericSymbolRef, bool) {
	ref := genericSymbolRefFromMatch(match)
	c.classify(&ref)
	if !jsFamilyReferenceClassVisible(ref.Class) {
		return genericSymbolRef{}, false
	}
	if !c.nameMatchRefMatchesDefinition(match, ref) {
		return genericSymbolRef{}, false
	}
	return ref, true
}

func (c *jsFamilyASTReferenceCollector) nameMatchRefMatchesDefinition(match genericSymbolMatch, ref genericSymbolRef) bool {
	switch ref.Class {
	case codeast.ClassImport:
		return c.importNameMatchRefMatchesDefinition(match)
	case jsast.ClassExport:
		return c.exportNameMatchRefMatchesDefinition(match)
	default:
		return !c.nameMatchUsesNonMatchingImportBinding(match, ref)
	}
}

func (c *jsFamilyASTReferenceCollector) importNameMatchRefMatchesDefinition(match genericSymbolMatch) bool {
	parsed := c.parsedFileForRefFile(match.File)
	if parsed == nil {
		return false
	}
	for _, binding := range jsFamilyImportBindingsWithParsed(parsed) {
		if !jsast.ImportBindingCoversLine(binding, match.Line) {
			continue
		}
		if c.importBindingMatchesDefinition(match.File, binding) {
			return true
		}
	}
	return false
}

func (c *jsFamilyASTReferenceCollector) exportNameMatchRefMatchesDefinition(match genericSymbolMatch) bool {
	parsed := c.parsedFileForRefFile(match.File)
	if parsed == nil {
		return false
	}
	sawExportBindingForSymbol := false
	for _, binding := range jsast.ExportBindingsWithParsed(parsed) {
		if binding.Line != match.Line || !jsFamilyExportBindingNamesSymbol(binding, c.symbol) {
			continue
		}
		sawExportBindingForSymbol = true
		if c.exportBindingMatchesDefinition(match.File, binding) {
			return true
		}
	}
	if sawExportBindingForSymbol {
		return false
	}
	return c.refFileMatchesDefinition(match.File) ||
		c.fileHasImportBindingLocalMatchingDefinition(match.File, c.symbol)
}

func (c *jsFamilyASTReferenceCollector) nameMatchUsesNonMatchingImportBinding(match genericSymbolMatch, ref genericSymbolRef) bool {
	parsed := c.parsedFileForRefFile(match.File)
	if parsed == nil {
		return false
	}
	for _, binding := range jsFamilyImportBindingsWithParsed(parsed) {
		if binding.Local != c.symbol {
			continue
		}
		usesBinding := false
		jsast.VisitImportBindingUsagesWithParsed(parsed, binding, func(usage jsast.ImportBindingUsage) bool {
			if usage.Line == match.Line && usage.Class == ref.Class {
				usesBinding = true
				return false
			}
			return true
		})
		if usesBinding {
			return !c.importBindingMatchesDefinition(match.File, binding)
		}
	}
	return false
}

func (c *jsFamilyASTReferenceCollector) importBindingRefFromSourceMatch(match genericSymbolMatch) (genericSymbolRef, bool) {
	return c.importBindingRefFromMatch(match, c.importBindingMatchesDefinition)
}

func (c *jsFamilyASTReferenceCollector) defaultImportBindingRefFromSourceMatch(match genericSymbolMatch) (genericSymbolRef, bool) {
	return c.importBindingRefFromMatch(match, func(importFile string, binding jsast.ImportBinding) bool {
		return jsFamilyImportBindingIsDefault(binding) && c.importBindingMatchesDefinition(importFile, binding)
	})
}

func (c *jsFamilyASTReferenceCollector) importBindingRefFromMatch(match genericSymbolMatch, matches func(string, jsast.ImportBinding) bool) (genericSymbolRef, bool) {
	parsed := c.parsedFileForRefFile(match.File)
	if parsed == nil {
		return genericSymbolRef{}, false
	}
	for _, binding := range jsFamilyImportBindingsWithParsed(parsed) {
		if !jsast.ImportBindingCoversLine(binding, match.Line) || !matches(match.File, binding) {
			continue
		}
		ref := genericSymbolRefFromMatch(match)
		ref.Class = codeast.ClassImport
		return ref, true
	}
	return genericSymbolRef{}, false
}

func (c *jsFamilyASTReferenceCollector) addRef(ref genericSymbolRef) bool {
	c.refs = append(c.refs, ref)
	if c.limit <= 0 || len(c.refs) < c.limit {
		return true
	}
	c.truncated = true
	c.budgetLimitHit = true
	return false
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
