package navigation

func findReferencesWithFallbackRuntime(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime, referenceFilter ReferenceFilter, fallbackSearchPath string) ([]Reference, bool, bool) {
	ambiguousFiles := findAmbiguousFilesWithRuntime(baseName, cand, runtime)
	allRefs, truncated, incomplete := findReferences(baseName, referenceFilter, fallbackSearchPath)
	return filterRefsByCandidate(allRefs, cand, ambiguousFiles), truncated, incomplete
}
