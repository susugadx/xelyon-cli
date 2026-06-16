package search

type genericLanguageResolver struct {
	lang string
}

func (r genericLanguageResolver) Resolve(symbol string, opts SearchOptions) symbolResolveResult {
	var result genericResolveResult
	switch r.lang {
	case "js":
		result = resolveJSFamilySymbol(symbol, opts)
	case "python":
		result = resolvePythonSymbol(symbol, opts)
	case "rust":
		result = resolveRustSymbol(symbol, opts)
	case "java":
		result = resolveJavaSymbol(symbol, opts)
	case "csharp":
		result = resolveCSharpSymbol(symbol, opts)
	case "php":
		result = resolvePHPSymbol(symbol, opts)
	case "ruby":
		result = resolveRubySymbol(symbol, opts)
	case "swift":
		result = resolveSwiftSymbol(symbol, opts)
	case "scala":
		result = resolveScalaSymbol(symbol, opts)
	case "elixir":
		result = resolveElixirSymbol(symbol, opts)
	case "lua":
		result = resolveLuaSymbol(symbol, opts)
	case "cpp":
		result = resolveCppSymbol(symbol, opts)
	default:
		result = resolveGenericSymbol(symbol, opts)
	}

	switch result.Status {
	case genericSymbolSingle:
		return symbolResolveResult{Output: result.Output, Status: symbolResolveSingle, Bundle: result.Bundle, Observation: result.Observation}
	case genericSymbolMultiple:
		return symbolResolveResult{Output: result.Output, Status: symbolResolveMultiple, AffectedFiles: result.AffectedFiles, Observation: result.Observation}
	default:
		return symbolResolveResult{Status: symbolResolveNone}
	}
}
