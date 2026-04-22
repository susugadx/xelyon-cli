package navigation

func findReferencesWithFallbackRuntime(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime) ([]Reference, bool, bool) {
	ambiguousFiles := findAmbiguousFilesWithRuntime(baseName, cand, runtime)
	allRefs, truncated, incomplete := findReferences(baseName)
	return filterRefsByCandidate(allRefs, cand, ambiguousFiles), truncated, incomplete
}
