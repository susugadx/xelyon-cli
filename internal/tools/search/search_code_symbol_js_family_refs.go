package search

const jsFamilyBundleLSPSource = "TypeScript/JavaScript LSP"

type jsFamilyReferenceResult struct {
	refs        []genericSymbolRef
	totalRefs   []genericSymbolRef
	diagnostics SymbolBundleDiagnostics
}

func (result jsFamilyReferenceResult) refsForTotals() []genericSymbolRef {
	if len(result.totalRefs) > 0 {
		return result.totalRefs
	}
	return result.refs
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

func setJSFamilyBundleDiagnostics(bundle *SymbolBundle, diagnostics SymbolBundleDiagnostics, totalRefs []genericSymbolRef) {
	if bundle == nil {
		return
	}
	diagnostics = cloneSymbolBundleDiagnostics(diagnostics)
	accepted := len(dedupeGenericRefs(totalRefs))
	raw := accepted
	if diagnostics.RawRefCount != nil {
		raw = *diagnostics.RawRefCount
	}
	updateDiagnosticsRefCounts(&diagnostics, raw, accepted)
	if diagnostics.ResolvedBy == symbolBundleResolvedByLSP {
		diagnostics.ResolvedViaLSP = true
		diagnostics.LSPSource = jsFamilyBundleLSPSource
	}
	bundle.Diagnostics = diagnostics
	finalizeSymbolBundleDiagnostics(bundle)
}

func findJSFamilyReferencesWithSemantic(symbol string, def genericSymbolDef, opts jsFamilyReferenceOptions) jsFamilyReferenceResult {
	if opts.lsp.request.LSPClient != nil && def.Character > 0 {
		collection, err := findJSFamilyReferencesWithLSP(symbol, def, opts.lsp)
		if err == nil && collection.hasRawLocations() {
			return jsFamilyReferenceResult{
				refs:        collection.refs,
				totalRefs:   collection.summaryRefs,
				diagnostics: jsFamilyLSPDiagnostics(collection),
			}
		}
		astResult := findJSFamilyReferencesWithASTDetailed(symbol, def, opts.nameOnly)
		return jsFamilyReferenceResult{
			refs:        astResult.refs,
			diagnostics: jsFamilyMixedFallbackDiagnostics(err, astResult, collection),
		}
	}
	astResult := findJSFamilyReferencesWithASTDetailed(symbol, def, opts.nameOnly)
	return jsFamilyReferenceResult{
		refs:        astResult.refs,
		diagnostics: jsFamilyASTDiagnostics(astResult),
	}
}

func jsFamilyLSPDiagnostics(collection jsFamilyLSPReferenceCollection) SymbolBundleDiagnostics {
	diagnostics := SymbolBundleDiagnostics{
		ResolvedBy:     symbolBundleResolvedByLSP,
		ResolvedViaLSP: true,
		LSPSource:      jsFamilyBundleLSPSource,
		LSPAttempted:   boolPtr(true),
		LSPAvailable:   boolPtr(true),
		LSPTimedOut:    boolPtr(false),
		FallbackUsed:   boolPtr(false),
		Incomplete:     boolPtr(false),
		Truncated:      boolPtr(false),
		BudgetLimitHit: boolPtr(collection.rawLocationCount > jsFamilyLSPReferenceEvidenceLimit),
	}
	updateDiagnosticsRefCounts(&diagnostics, collection.rawLocationCount, len(collection.summaryRefs))
	diagnostics.Confidence = inferSymbolBundleConfidence(diagnostics)
	normalizeSymbolBundleDiagnostics(&diagnostics)
	return diagnostics
}

func jsFamilyASTDiagnostics(result jsFamilyASTReferenceResult) SymbolBundleDiagnostics {
	diagnostics := SymbolBundleDiagnostics{
		ResolvedBy:     symbolBundleResolvedByAST,
		LSPAttempted:   boolPtr(false),
		LSPAvailable:   boolPtr(false),
		LSPTimedOut:    boolPtr(false),
		FallbackUsed:   boolPtr(false),
		Incomplete:     boolPtr(result.incomplete),
		Truncated:      boolPtr(result.truncated),
		BudgetLimitHit: boolPtr(result.budgetLimitHit),
	}
	updateDiagnosticsRefCounts(&diagnostics, result.rawMatchCount, len(result.refs))
	diagnostics.Confidence = inferSymbolBundleConfidence(diagnostics)
	normalizeSymbolBundleDiagnostics(&diagnostics)
	return diagnostics
}

func jsFamilyMixedFallbackDiagnostics(err error, result jsFamilyASTReferenceResult, collection jsFamilyLSPReferenceCollection) SymbolBundleDiagnostics {
	diagnostics := jsFamilyASTDiagnostics(result)
	diagnostics.ResolvedBy = symbolBundleResolvedByMixed
	diagnostics.LSPAttempted = boolPtr(true)
	diagnostics.FallbackUsed = boolPtr(true)
	diagnostics.LSPSource = jsFamilyBundleLSPSource
	if err == nil {
		diagnostics.LSPAvailable = boolPtr(true)
		diagnostics.FallbackReason = symbolBundleFallbackReasonLSPEmpty
	} else if lspReferenceErrorTimedOutForSearch(err) {
		diagnostics.LSPAvailable = boolPtr(false)
		diagnostics.LSPTimedOut = boolPtr(true)
		diagnostics.FallbackReason = symbolBundleFallbackReasonLSPTimeout
	} else {
		diagnostics.LSPAvailable = boolPtr(false)
		diagnostics.FallbackReason = symbolBundleFallbackReasonLSPError
	}
	if collection.rawLocationCount > 0 && diagnostics.RawRefCount != nil {
		raw := collection.rawLocationCount + *diagnostics.RawRefCount
		diagnostics.RawRefCount = intPtr(raw)
	}
	diagnostics.Confidence = inferSymbolBundleConfidence(diagnostics)
	normalizeSymbolBundleDiagnostics(&diagnostics)
	return diagnostics
}
