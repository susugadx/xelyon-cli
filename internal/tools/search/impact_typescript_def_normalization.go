package search

import (
	"regexp"
	"strings"
)

func normalizeStructuredTypeScriptDefForImpact(def genericSymbolDef) genericSymbolDef {
	target, ok := structuredTypeScriptImplementationTargetForPath(def.File)
	if !ok || target.suffix != structuredTypeScriptTSXImpactTarget.suffix {
		return def
	}
	if !isStructuredTypeScriptFunctionExpressionComponentSignature(def.Signature, def.Name) {
		return def
	}
	def.Kind = "function"
	return def
}

func isStructuredTypeScriptFunctionExpressionComponentSignature(signature string, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	pattern := `^(?:export\s+)?const\s+` + regexp.QuoteMeta(name) + `\b(?:\s*:[^=]+)?\s*=\s*(?:async\s+)?function(?:\s+` + regexp.QuoteMeta(name) + `\b|\s*\()`
	return regexp.MustCompile(pattern).MatchString(strings.TrimSpace(signature))
}
