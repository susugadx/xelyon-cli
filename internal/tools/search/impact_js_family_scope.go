package search

func newJSFamilyStructuredImpactReferenceOptions(def genericSymbolDef, opts SearchOptions, fileType string) jsFamilyReferenceOptions {
	semanticFilter := structuredImpactSemanticReferenceFilterOptions(opts)
	nameOnly := structuredImpactNameOnlyEvidenceOptions(def.File, opts)
	if fileType != "" {
		semanticFilter = structuredImpactEvidenceFileTypeOptions(semanticFilter, fileType)
		nameOnly = structuredImpactEvidenceFileTypeOptions(nameOnly, fileType)
	}
	return jsFamilyReferenceOptions{
		lsp:      newJSFamilyLSPReferenceOptions(opts, semanticFilter, opts),
		nameOnly: nameOnly,
	}
}
