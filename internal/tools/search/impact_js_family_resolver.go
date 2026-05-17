package search

type jsFamilyImpactDefinitionSet struct {
	defs                 []genericSymbolDef
	suppressedRefDefs    []genericSymbolDef
	definitionIncomplete bool
}

type jsFamilyImpactResolverSpec struct {
	findDefinitions         func(symbol string, opts SearchOptions) jsFamilyImpactDefinitionSet
	collectDefAffectedFiles func(defs []genericSymbolDef, opts SearchOptions) []string
	referenceOptions        func(def genericSymbolDef, opts SearchOptions) jsFamilyReferenceOptions
	normalizeRefs           func(refs []genericSymbolRef) []genericSymbolRef
	filterRefs              func(def genericSymbolDef, defs jsFamilyImpactDefinitionSet, refs []genericSymbolRef) []genericSymbolRef
	buildBundle             func(symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef, totalRefs []genericSymbolRef) *SymbolBundle
}

func resolveStructuredJSFamilyImpactSymbol(symbol string, scope structuredImpactScope, spec jsFamilyImpactResolverSpec) symbolResolveResult {
	if !spec.valid() {
		return symbolResolveResult{Status: symbolResolveNone}
	}

	opts := scope.Definition
	definitionSet := spec.findDefinitions(symbol, opts)
	defs := definitionSet.defs
	if len(defs) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	if shouldDeferIncompleteJSFamilyDefinitions(definitionSet.definitionIncomplete) {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	if len(defs) > 1 {
		return symbolResolveResult{
			Output:        formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts),
			Status:        symbolResolveMultiple,
			AffectedFiles: spec.collectDefAffectedFiles(defs, opts),
		}
	}

	def := defs[0]
	refOpts := spec.referenceOptions(def, scope.Evidence)
	refResult := findJSFamilyReferencesWithSemantic(symbol, def, refOpts)
	refs := spec.normalizeRefs(refResult.refs)
	totalRefs := spec.normalizeRefs(refResult.refsForTotals())
	if spec.filterRefs != nil {
		refs = spec.filterRefs(def, definitionSet, refs)
		totalRefs = spec.filterRefs(def, definitionSet, totalRefs)
	}
	refs = filterGenericRefs(refs, def)
	totalRefs = filterGenericRefs(totalRefs, def)

	bundle := spec.buildBundle(symbol, def, refOpts.nameOnly, refs, totalRefs)
	if bundle == nil || bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	setJSFamilyBundleLSPDiagnostics(bundle, refResult.resolvedViaLSP)

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
}

func (spec jsFamilyImpactResolverSpec) valid() bool {
	return spec.findDefinitions != nil &&
		spec.collectDefAffectedFiles != nil &&
		spec.referenceOptions != nil &&
		spec.normalizeRefs != nil &&
		spec.buildBundle != nil
}
