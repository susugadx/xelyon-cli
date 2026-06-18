package search

type genericLanguageResolveFunc func(symbol string, opts SearchOptions) genericResolveResult

type genericLanguageResolverSpec struct {
	language string
	resolve  genericLanguageResolveFunc
}

type genericLanguageResolver struct {
	spec genericLanguageResolverSpec
}

func (r genericLanguageResolver) Resolve(symbol string, opts SearchOptions) symbolResolveResult {
	result := r.resolve(symbol, opts)

	switch result.Status {
	case genericSymbolSingle:
		return symbolResolveResult{Output: result.Output, Status: symbolResolveSingle, Bundle: result.Bundle, Observation: result.Observation}
	case genericSymbolMultiple:
		return symbolResolveResult{Output: result.Output, Status: symbolResolveMultiple, AffectedFiles: result.AffectedFiles, Observation: result.Observation}
	default:
		return symbolResolveResult{Status: symbolResolveNone}
	}
}

func (r genericLanguageResolver) resolve(symbol string, opts SearchOptions) genericResolveResult {
	if r.spec.resolve != nil {
		return r.spec.resolve(symbol, opts)
	}
	return resolveGenericSymbol(symbol, opts)
}
