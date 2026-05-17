package search

import (
	"fmt"
	"regexp"
	"strings"
)

func findStructuredJavaScriptImpactDefinitions(symbol string, opts SearchOptions) []genericSymbolDef {
	defs := normalizeStructuredJavaScriptDefs(findJSFamilyDefinitionsWithAST(symbol, opts))
	if len(defs) == 0 {
		defs = normalizeStructuredJavaScriptDefs(findGenericDefinitions(symbol, opts))
	}
	defs = append(defs, findStructuredJavaScriptCommonJSInlineDefinitions(symbol, opts)...)
	return dedupeStructuredJavaScriptDefs(defs)
}

func findStructuredJavaScriptImpactDefinitionSet(symbol string, opts SearchOptions) jsFamilyImpactDefinitionSet {
	return jsFamilyImpactDefinitionSet{defs: findStructuredJavaScriptImpactDefinitions(symbol, opts)}
}

func normalizeStructuredJavaScriptDefForImpact(def genericSymbolDef) genericSymbolDef {
	if isStructuredJavaScriptFunctionExpressionSignature(def.Signature, def.Name) {
		def.Kind = "function"
	}
	return def
}

func isStructuredJavaScriptFunctionExpressionSignature(signature string, name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}

	pattern := `^(?:export\s+)?(?:const|let)\s+` + regexp.QuoteMeta(name) + `\b\s*=\s*(?:async\s+)?function(?:\s+` + regexp.QuoteMeta(name) + `\b|\s*\()`
	return regexp.MustCompile(pattern).MatchString(strings.TrimSpace(signature))
}

func findStructuredJavaScriptCommonJSInlineDefinitions(symbol string, opts SearchOptions) []genericSymbolDef {
	matches := findGenericSymbolMatches(symbol, opts, 0)
	defs := make([]genericSymbolDef, 0)
	for _, match := range matches {
		if !isJavaScriptSourceFilePath(match.File) {
			continue
		}
		if name, kind, ok := parseJavaScriptCommonJSInlineDefinition(match.Content, symbol); ok {
			defs = append(defs, genericSymbolDef{
				Name:      name,
				Kind:      kind,
				File:      cleanStructuredJavaScriptDisplayPath(match.File),
				Line:      match.Line,
				Signature: strings.TrimSpace(match.Content),
			})
		}
	}
	return defs
}

func parseJavaScriptCommonJSInlineDefinition(signature string, symbol string) (string, string, bool) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return "", "", false
	}

	signature = strings.TrimSpace(signature)
	for _, parser := range []func(string, string) (string, string, bool){
		parseJavaScriptModuleExportsInlineDefinition,
		parseJavaScriptNamedCommonJSInlineDefinition,
	} {
		if name, kind, ok := parser(signature, symbol); ok {
			return name, kind, true
		}
	}
	return "", "", false
}

func parseJavaScriptModuleExportsInlineDefinition(signature string, symbol string) (string, string, bool) {
	pattern := `^module\s*\.\s*exports\s*=\s*(?:async\s+)?function\s+` + regexp.QuoteMeta(symbol) + `\b`
	if regexp.MustCompile(pattern).MatchString(signature) {
		return symbol, "function", true
	}
	pattern = `^module\s*\.\s*exports\s*=\s*class\s+` + regexp.QuoteMeta(symbol) + `\b`
	if regexp.MustCompile(pattern).MatchString(signature) {
		return symbol, "class", true
	}
	return "", "", false
}

func parseJavaScriptNamedCommonJSInlineDefinition(signature string, symbol string) (string, string, bool) {
	exportTarget := `(?:exports|module\s*\.\s*exports)\s*(?:\.\s*` + regexp.QuoteMeta(symbol) + `|\[\s*['"]` + regexp.QuoteMeta(symbol) + `['"]\s*\])\s*=\s*`
	if regexp.MustCompile(`^` + exportTarget + `(?:async\s+)?function(?:\s+` + regexp.QuoteMeta(symbol) + `\b|\s*\()`).MatchString(signature) {
		return symbol, "function", true
	}
	if regexp.MustCompile(`^` + exportTarget + `class(?:\s+` + regexp.QuoteMeta(symbol) + `\b|\s*\{)`).MatchString(signature) {
		return symbol, "class", true
	}
	return "", "", false
}

func dedupeStructuredJavaScriptDefs(defs []genericSymbolDef) []genericSymbolDef {
	seen := make(map[string]struct{}, len(defs))
	result := make([]genericSymbolDef, 0, len(defs))
	for _, def := range defs {
		key := fmt.Sprintf("%s:%d:%s:%s", def.File, def.Line, def.Name, def.Kind)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, def)
	}
	return result
}

func collectStructuredJavaScriptDefAffectedFiles(defs []genericSymbolDef, opts SearchOptions) []string {
	rootPath := structuredJavaScriptImpactFileRoot(opts)
	paths := make([]string, 0, len(defs))
	for _, def := range defs {
		if absPath := absoluteAffectedFilePathWithBase(def.File, rootPath); absPath != "" {
			paths = append(paths, absPath)
		}
	}
	return dedupePaths(paths)
}
