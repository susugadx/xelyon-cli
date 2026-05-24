package search

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/jsast"
)

type jsFamilyASTImportBindingCompletionKey struct {
	file string
	name string
}

func (c *jsFamilyASTReferenceCollector) recordImportBindingImport(importRef genericSymbolRef) {
	c.importBindingImports = append(c.importBindingImports, importRef)
}

func (c *jsFamilyASTReferenceCollector) refsWithImportBindingUsages() []genericSymbolRef {
	c.expandImportBindingUsages()
	if len(c.importBindingRefs) == 0 {
		return c.refs
	}
	refs := make([]genericSymbolRef, 0, len(c.refs)+len(c.importBindingRefs))
	refs = append(refs, c.refs...)
	refs = append(refs, c.importBindingRefs...)
	return refs
}

func (c *jsFamilyASTReferenceCollector) expandImportBindingUsages() {
	if c.importBindingsExpanded {
		return
	}
	c.importBindingsExpanded = true
	for _, importRef := range c.importBindingImports {
		if !c.addImportBindingUsages(importRef) {
			return
		}
	}
}

func (c *jsFamilyASTReferenceCollector) addImportBindingUsages(importRef genericSymbolRef) bool {
	parsed := c.parsedFileForRefFile(importRef.File)
	if parsed == nil {
		return true
	}
	for _, binding := range jsFamilyImportBindingsWithParsed(parsed) {
		if !c.importBindingMatchesDefinition(importRef.File, binding) {
			continue
		}
		if !c.markImportBindingCompletion(importRef.File, binding.Local) {
			continue
		}
		keepScanning := true
		jsast.VisitImportBindingUsagesWithParsed(parsed, binding, func(usage jsast.ImportBindingUsage) bool {
			if !jsFamilyImportBindingUsageVisibleForBinding(binding, usage) {
				return true
			}
			ref := genericSymbolRef{
				File:    importRef.File,
				Line:    usage.Line,
				Snippet: usage.Snippet,
				IsTest:  importRef.IsTest,
				Class:   usage.Class,
			}
			keepScanning = c.addImportBindingRef(ref)
			return keepScanning
		})
		if !keepScanning {
			return false
		}
	}
	return true
}

func (c *jsFamilyASTReferenceCollector) importBindingMatchesDefinition(importFile string, binding jsast.ImportBinding) bool {
	if !c.importBindingSourceMatchesDefinition(importFile, binding.Source) {
		return false
	}
	switch binding.Kind {
	case jsast.ImportBindingNamed:
		if binding.Imported == "default" {
			return c.definitionAllowsDefaultImport() && binding.Local != ""
		}
		return binding.Imported == c.symbol && binding.Local != ""
	case jsast.ImportBindingDefault:
		return c.definitionAllowsDefaultImport() && binding.Local != ""
	case jsast.ImportBindingType:
		if binding.Imported == "default" {
			return c.definitionAllowsDefaultImport() && binding.Local != ""
		}
		return binding.Imported == c.symbol && binding.Local != ""
	default:
		return false
	}
}

func (c *jsFamilyASTReferenceCollector) exportBindingMatchesDefinition(exportFile string, binding jsast.ExportBinding) bool {
	if !jsFamilyExportBindingNamesSymbol(binding, c.symbol) {
		return false
	}
	if strings.TrimSpace(binding.Source) != "" {
		return c.importBindingSourceMatchesDefinition(exportFile, binding.Source)
	}
	return c.refFileMatchesDefinition(exportFile) ||
		c.fileHasImportBindingLocalMatchingDefinition(exportFile, binding.Local)
}

func jsFamilyExportBindingNamesSymbol(binding jsast.ExportBinding, symbol string) bool {
	return binding.Local == symbol || binding.Exported == symbol
}

func (c *jsFamilyASTReferenceCollector) fileHasImportBindingLocalMatchingDefinition(importFile string, localName string) bool {
	parsed := c.parsedFileForRefFile(importFile)
	if parsed == nil || localName == "" {
		return false
	}
	for _, binding := range jsFamilyImportBindingsWithParsed(parsed) {
		if binding.Local == localName && c.importBindingMatchesDefinition(importFile, binding) {
			return true
		}
	}
	return false
}

func jsFamilyImportBindingIsDefault(binding jsast.ImportBinding) bool {
	return binding.Kind == jsast.ImportBindingDefault ||
		(binding.Kind == jsast.ImportBindingNamed && binding.Imported == "default") ||
		(binding.Kind == jsast.ImportBindingType && binding.Imported == "default")
}

func jsFamilyImportBindingsWithParsed(parsed *jsast.ParsedFile) []jsast.ImportBinding {
	bindings := jsast.ImportBindingsWithParsed(parsed)
	bindings = append(bindings, jsast.TypeImportBindingsWithParsed(parsed)...)
	return append(bindings, jsast.RequireBindingsWithParsed(parsed)...)
}

func jsFamilyImportBindingUsageVisibleForBinding(binding jsast.ImportBinding, usage jsast.ImportBindingUsage) bool {
	if binding.Kind == jsast.ImportBindingType {
		return usage.Class == jsast.ClassTypeRef
	}
	return true
}

func (c *jsFamilyASTReferenceCollector) importBindingSourceMatchesDefinition(importFile string, source string) bool {
	source = strings.TrimSpace(source)
	if source == "" || !strings.HasPrefix(source, ".") {
		return false
	}

	defAbs := c.definitionAbsPath()
	importAbs := absoluteAffectedFilePath(importFile, c.opts, affectedFileSourceText)
	if defAbs == "" || importAbs == "" {
		return false
	}
	sourceBase := filepath.Clean(filepath.Join(filepath.Dir(importAbs), filepath.FromSlash(source)))
	return jsFamilyImportSourceBaseMatchesFile(sourceBase, defAbs)
}

func (c *jsFamilyASTReferenceCollector) refFileMatchesDefinition(file string) bool {
	defAbs := c.definitionAbsPath()
	refAbs := absoluteAffectedFilePath(file, c.opts, affectedFileSourceText)
	if defAbs == "" || refAbs == "" {
		return false
	}
	return filepath.Clean(defAbs) == filepath.Clean(refAbs)
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

func (c *jsFamilyASTReferenceCollector) definitionAllowsDefaultImport() bool {
	if jsFamilyDefinitionSignatureAllowsDefaultImport(c.def.Signature) {
		return true
	}
	abs := c.definitionAbsPath()
	if abs == "" || c.def.Line <= 0 {
		return false
	}
	if parsed := c.files.Parsed(abs); parsed != nil && jsast.SymbolExportedAsDefaultWithParsed(parsed, c.symbol) {
		return true
	}
	src, err := os.ReadFile(abs)
	if err != nil {
		return false
	}
	lines := strings.Split(string(src), "\n")
	idx := c.def.Line - 1
	return idx >= 0 && idx < len(lines) && strings.Contains(lines[idx], "export default")
}

func jsFamilyDefinitionSignatureAllowsDefaultImport(signature string) bool {
	signature = strings.TrimSpace(signature)
	return strings.HasPrefix(signature, "export default ") ||
		strings.HasPrefix(signature, "module.exports =")
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

func (c *jsFamilyASTReferenceCollector) addImportBindingRef(ref genericSymbolRef) bool {
	c.aliasUsageCount++
	c.importBindingRefs = append(c.importBindingRefs, ref)
	if c.limit <= 0 || len(c.importBindingRefs) < c.limit {
		return true
	}
	c.truncated = true
	c.budgetLimitHit = true
	return false
}

func (c *jsFamilyASTReferenceCollector) markImportBindingCompletion(file string, localName string) bool {
	if c.completedImportBindings == nil {
		c.completedImportBindings = make(map[jsFamilyASTImportBindingCompletionKey]struct{})
	}
	key := jsFamilyASTImportBindingCompletionKey{file: file, name: localName}
	if _, ok := c.completedImportBindings[key]; ok {
		return false
	}
	c.completedImportBindings[key] = struct{}{}
	return true
}
