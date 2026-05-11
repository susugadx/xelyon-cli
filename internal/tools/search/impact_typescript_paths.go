package search

import (
	"fmt"
	"path/filepath"
	"strings"
)

func cleanStructuredTypeScriptFilePattern(pattern string) string {
	return filepath.ToSlash(strings.TrimSpace(pattern))
}

func isTypeScriptSourceFilePath(path string) bool {
	_, ok := structuredTypeScriptSourceTargetForPath(path)
	return ok
}

func isTypeScriptDeclarationFilePath(path string) bool {
	_, ok := structuredTypeScriptDeclarationTargetForPath(path)
	return ok
}

func isTypeScriptImplementationFilePath(path string) bool {
	_, ok := structuredTypeScriptImplementationTargetForPath(path)
	return ok
}

func isTypeScriptOnlyFilePattern(pattern string) bool {
	return structuredTypeScriptImpactAllowsFilePattern(pattern)
}

func normalizeStructuredTypeScriptDefs(defs []genericSymbolDef) []genericSymbolDef {
	normalized := make([]genericSymbolDef, len(defs))
	for i, def := range defs {
		normalized[i] = def
		normalized[i].File = cleanStructuredTypeScriptDisplayPath(def.File)
		normalized[i] = normalizeStructuredTypeScriptDefForImpact(normalized[i])
	}
	return normalized
}

func normalizeStructuredTypeScriptRefs(refs []genericSymbolRef) []genericSymbolRef {
	normalized := make([]genericSymbolRef, len(refs))
	for i, ref := range refs {
		normalized[i] = ref
		normalized[i].File = cleanStructuredTypeScriptDisplayPath(ref.File)
	}
	return normalized
}

func structuredTypeScriptDefPathKey(path string) string {
	return cleanStructuredTypeScriptDisplayPath(path)
}

func structuredTypeScriptLocationKey(file string, line int) string {
	if file == "" || line <= 0 {
		return ""
	}
	return fmt.Sprintf("%s:%d", file, line)
}

func cleanStructuredTypeScriptDisplayPath(path string) string {
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

func structuredTypeScriptImpactFileRoot(opts SearchOptions) string {
	basis := resolveSearchPathBasisForOptions(opts)
	if strings.TrimSpace(basis.MatchRoot) != "" {
		return basis.MatchRoot
	}
	if strings.TrimSpace(basis.Workdir) != "" {
		return basis.Workdir
	}
	return invocationCWDOrGetwd(opts)
}
