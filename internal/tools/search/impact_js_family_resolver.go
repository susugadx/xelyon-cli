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
	language                string
	rootPath                func(opts SearchOptions) string
	debugSource             func(def genericSymbolDef) string
	buildSemanticEvidence   jsFamilySemanticEvidenceBuilder
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

	diagnostics := normalizedJSFamilyBundleDiagnostics(refResult.diagnostics, totalRefs)
	evidence, ok := spec.buildSemanticEvidence(spec.language, symbol, def, refOpts.nameOnly, refs, totalRefs, diagnostics)
	if !ok {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	bundle, _ := buildSymbolBundleFromSemanticEvidence(evidence)
	if bundle == nil || bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	applyJSFamilySemanticImpactBundleDebug(bundle, spec, def, refOpts.nameOnly)

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
		spec.rootPath != nil &&
		spec.debugSource != nil &&
		spec.buildSemanticEvidence != nil
}

func applyJSFamilySemanticImpactBundleDebug(bundle *SymbolBundle, spec jsFamilyImpactResolverSpec, def genericSymbolDef, opts SearchOptions) {
	if bundle == nil {
		return
	}
	bundle.Debug.Source = spec.debugSource(def)
	bundle.Debug.FileRootPath = spec.rootPath(opts)
}
