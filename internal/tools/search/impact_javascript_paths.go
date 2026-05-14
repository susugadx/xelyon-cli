package search

import (
	"path/filepath"
	"strings"
)

func cleanStructuredJavaScriptFilePattern(pattern string) string {
	return filepath.ToSlash(strings.TrimSpace(pattern))
}

func isJavaScriptSourceFilePath(path string) bool {
	path = strings.ToLower(cleanStructuredJavaScriptDisplayPath(path))
	return strings.HasSuffix(path, ".js") && !strings.HasSuffix(path, ".d.ts")
}

func isJavaScriptOnlyFilePattern(pattern string) bool {
	pattern = strings.ToLower(cleanStructuredJavaScriptFilePattern(pattern))
	return strings.HasSuffix(pattern, ".js") && !strings.HasSuffix(pattern, ".jsx") && !strings.HasSuffix(pattern, ".mjs") && !strings.HasSuffix(pattern, ".cjs")
}

func normalizeStructuredJavaScriptDefs(defs []genericSymbolDef) []genericSymbolDef {
	normalized := make([]genericSymbolDef, len(defs))
	for i, def := range defs {
		normalized[i] = def
		normalized[i].File = cleanStructuredJavaScriptDisplayPath(def.File)
		normalized[i] = normalizeStructuredJavaScriptDefForImpact(normalized[i])
	}
	return normalized
}

func normalizeStructuredJavaScriptRefs(refs []genericSymbolRef) []genericSymbolRef {
	normalized := make([]genericSymbolRef, len(refs))
	for i, ref := range refs {
		normalized[i] = ref
		normalized[i].File = cleanStructuredJavaScriptDisplayPath(ref.File)
	}
	return normalized
}

func cleanStructuredJavaScriptDisplayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	clean := filepath.ToSlash(filepath.Clean(path))
	if clean == "." {
		return ""
	}
	return clean
}

func structuredJavaScriptImpactFileRoot(opts SearchOptions) string {
	basis := resolveSearchPathBasisForOptions(opts)
	if strings.TrimSpace(basis.MatchRoot) != "" {
		return basis.MatchRoot
	}
	if strings.TrimSpace(basis.Workdir) != "" {
		return basis.Workdir
	}
	return invocationCWDOrGetwd(opts)
}

func structuredJavaScriptImpactReferenceOptions(def genericSymbolDef, opts SearchOptions) jsFamilyReferenceOptions {
	return newJSFamilyStructuredImpactReferenceOptions(def, opts, "js")
}
