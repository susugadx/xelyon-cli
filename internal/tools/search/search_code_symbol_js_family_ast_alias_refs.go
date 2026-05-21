package search

import (
	"path/filepath"
	"strings"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/jsast"
)

type jsFamilyASTAliasCompletionKey struct {
	file string
	name string
}

func (c *jsFamilyASTReferenceCollector) recordJSXAliasImport(importRef genericSymbolRef) {
	c.aliasImports = append(c.aliasImports, importRef)
}

func (c *jsFamilyASTReferenceCollector) refsWithJSXAliasUsages() []genericSymbolRef {
	c.expandJSXAliasImportUsages()
	if len(c.aliasRefs) == 0 {
		return c.refs
	}
	refs := make([]genericSymbolRef, 0, len(c.refs)+len(c.aliasRefs))
	refs = append(refs, c.refs...)
	refs = append(refs, c.aliasRefs...)
	return refs
}

func (c *jsFamilyASTReferenceCollector) expandJSXAliasImportUsages() {
	if c.aliasesExpanded {
		return
	}
	c.aliasesExpanded = true
	for _, importRef := range c.aliasImports {
		if !c.addJSXAliasImportUsages(importRef) {
			return
		}
	}
}

func (c *jsFamilyASTReferenceCollector) addJSXAliasImportUsages(importRef genericSymbolRef) bool {
	parsed := c.parsedFileForRefFile(importRef.File)
	if parsed == nil {
		return true
	}
	for _, alias := range jsast.NamedImportAliasesWithParsed(parsed, c.symbol) {
		if !c.aliasImportSourceMatchesDefinition(importRef, alias) {
			continue
		}
		if !c.markJSXAliasCompletion(importRef.File, alias.Local) {
			continue
		}
		keepScanning := true
		jsast.VisitJSXLocalNameUsagesForNamedImportAliasWithParsed(parsed, alias, func(usage jsast.JSXLocalNameUsage) bool {
			ref := genericSymbolRef{
				File:    importRef.File,
				Line:    usage.Line,
				Snippet: usage.Snippet,
				IsTest:  importRef.IsTest,
				Class:   codeast.ClassCall,
			}
			keepScanning = c.addAliasRef(ref)
			return keepScanning
		})
		if !keepScanning {
			return false
		}
	}
	return true
}

func (c *jsFamilyASTReferenceCollector) aliasImportSourceMatchesDefinition(importRef genericSymbolRef, alias jsast.NamedImportAlias) bool {
	source := strings.TrimSpace(alias.Source)
	if source == "" || !strings.HasPrefix(source, ".") {
		return false
	}

	defAbs := c.definitionAbsPath()
	importAbs := absoluteAffectedFilePath(importRef.File, c.opts, affectedFileSourceText)
	if defAbs == "" || importAbs == "" {
		return false
	}
	sourceBase := filepath.Clean(filepath.Join(filepath.Dir(importAbs), filepath.FromSlash(source)))
	return jsFamilyImportSourceBaseMatchesFile(sourceBase, defAbs)
}

func (c *jsFamilyASTReferenceCollector) definitionAbsPath() string {
	if c.def.File == "" {
		return ""
	}
	if abs := absoluteAffectedFilePath(c.def.File, c.opts, affectedFileSourceText); abs != "" {
		return abs
	}
	return absoluteAffectedFilePathWithBase(c.def.File, invocationCWDOrGetwd(c.opts))
}

func jsFamilyImportSourceBaseMatchesFile(sourceBase string, file string) bool {
	sourceBase = filepath.Clean(sourceBase)
	file = filepath.Clean(file)
	if sourceBase == file {
		return true
	}
	for _, candidate := range jsFamilyImportSourceCandidateFiles(sourceBase) {
		if filepath.Clean(candidate) == file {
			return true
		}
	}
	return false
}

func jsFamilyImportSourceCandidateFiles(sourceBase string) []string {
	stems := jsFamilyImportSourceCandidateStems(sourceBase)
	extensions := []string{".ts", ".tsx", ".js", ".jsx", ".mjs", ".mts", ".cjs", ".cts", ".d.ts"}
	candidates := make([]string, 0, len(stems)*len(extensions)*2)
	for _, stem := range stems {
		for _, ext := range extensions {
			candidates = append(candidates, stem+ext)
			candidates = append(candidates, filepath.Join(stem, "index"+ext))
		}
	}
	return candidates
}

func jsFamilyImportSourceCandidateStems(sourceBase string) []string {
	stems := []string{sourceBase}
	for _, ext := range []string{".js", ".jsx", ".mjs", ".cjs"} {
		if strings.HasSuffix(sourceBase, ext) {
			stems = append(stems, strings.TrimSuffix(sourceBase, ext))
			break
		}
	}
	return stems
}

func (c *jsFamilyASTReferenceCollector) addAliasRef(ref genericSymbolRef) bool {
	c.aliasRefs = append(c.aliasRefs, ref)
	return c.limit <= 0 || len(c.aliasRefs) < c.limit
}

func (c *jsFamilyASTReferenceCollector) markJSXAliasCompletion(file string, localName string) bool {
	if c.completedAliases == nil {
		c.completedAliases = make(map[jsFamilyASTAliasCompletionKey]struct{})
	}
	key := jsFamilyASTAliasCompletionKey{file: file, name: localName}
	if _, ok := c.completedAliases[key]; ok {
		return false
	}
	c.completedAliases[key] = struct{}{}
	return true
}
