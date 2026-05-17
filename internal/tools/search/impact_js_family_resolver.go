package search

type jsFamilyImpactDefinitionSet struct {
	defs              []genericSymbolDef
	suppressedRefDefs []genericSymbolDef
}

type jsFamilyImpactResolverSpec struct {
	findDefinitions         func(symbol string, opts SearchOptions) jsFamilyImpactDefinitionSet
	collectDefAffectedFiles func(defs []genericSymbolDef, opts SearchOptions) []string
	referenceOptions        func(def genericSymbolDef, opts SearchOptions) jsFamilyReferenceOptions
	normalizeRefs           func(refs []genericSymbolRef) []genericSymbolRef
	filterRefs              func(def genericSymbolDef, defs jsFamilyImpactDefinitionSet, refs []genericSymbolRef) []genericSymbolRef
	buildBundle             func(symbol string, def genericSymbolDef, opts SearchOptions, refs []genericSymbolRef) *SymbolBundle
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
	if spec.filterRefs != nil {
		refs = spec.filterRefs(def, definitionSet, refs)
	}
	refs = filterGenericRefs(refs, def)

	bundle := spec.buildBundle(symbol, def, refOpts.nameOnly, refs)
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
