package search

const jsFamilyBundleLSPSource = "TypeScript/JavaScript LSP"

type jsFamilyReferenceResult struct {
	refs           []genericSymbolRef
	resolvedViaLSP bool
}

type jsFamilyReferenceOptions struct {
	lsp      jsFamilyLSPReferenceOptions
	nameOnly SearchOptions
}

type jsFamilyLSPReferenceOptions struct {
	request  SearchOptions
	filter   SearchOptions
	location jsFamilyLSPLocationOptions
}

type jsFamilyLSPLocationOptions struct {
	adapterBase string
	displayRoot string
}

func newJSFamilyReferenceOptions(opts SearchOptions) jsFamilyReferenceOptions {
	return jsFamilyReferenceOptions{
		lsp:      newJSFamilyLSPReferenceOptions(opts, opts, opts),
		nameOnly: opts,
	}
}

func newJSFamilyLSPReferenceOptions(request SearchOptions, filter SearchOptions, locationBase SearchOptions) jsFamilyLSPReferenceOptions {
	return jsFamilyLSPReferenceOptions{
		request:  request,
		filter:   filter,
		location: newJSFamilyLSPLocationOptions(locationBase),
	}
}

func newJSFamilyLSPLocationOptions(opts SearchOptions) jsFamilyLSPLocationOptions {
	return jsFamilyLSPLocationOptions{
		adapterBase: invocationCWDOrGetwd(opts),
		displayRoot: affectedFileBasePath(opts, affectedFileSourceText),
	}
}

func setJSFamilyBundleLSPDiagnostics(bundle *SymbolBundle, resolved bool) {
	if bundle == nil {
		return
	}
	bundle.Diagnostics.ResolvedViaLSP = resolved
	if resolved {
		bundle.Diagnostics.LSPSource = jsFamilyBundleLSPSource
	}
}

func findJSFamilyReferencesWithSemantic(symbol string, def genericSymbolDef, opts jsFamilyReferenceOptions) jsFamilyReferenceResult {
	if opts.lsp.request.LSPClient != nil && def.Character > 0 {
		refs, err := findJSFamilyReferencesWithLSP(symbol, def, opts.lsp)
		if err == nil && len(refs) > 0 {
			return jsFamilyReferenceResult{refs: refs, resolvedViaLSP: true}
		}
	}
	return jsFamilyReferenceResult{refs: findJSFamilyReferencesWithAST(symbol, opts.nameOnly)}
}
