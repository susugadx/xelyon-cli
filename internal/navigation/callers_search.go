package navigation

func findReferencesWithFallbackRuntime(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime, referenceFilter ReferenceFilter) ([]Reference, bool, bool) {
	ambiguousFiles := findAmbiguousFilesWithRuntime(baseName, cand, runtime)
	allRefs, truncated, incomplete := findReferences(baseName, referenceFilter)
	return filterRefsByCandidate(allRefs, cand, ambiguousFiles), truncated, incomplete
}
