package navigation

func findReferencesWithFallbackRuntime(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime, referenceFilter ReferenceFilter, fallbackSearchPath string) ([]Reference, bool, bool) {
	ambiguousFiles := findAmbiguousFilesWithRuntimePath(baseName, cand, runtime, fallbackSearchPath)
	allRefs, truncated, incomplete := findReferences(baseName, referenceFilter, fallbackSearchPath)
	allRefs = normalizeReferencesForCandidateRoot(allRefs, cand, runtime)
	return filterRefsByCandidate(allRefs, cand, ambiguousFiles), truncated, incomplete
}

func normalizeReferencesForCandidateRoot(refs []Reference, cand SymbolCandidate, runtime GoSymbolRuntime) []Reference {
	targetRoot := normalizeNavigationRootPath(cand.RootPath)
	if targetRoot == "" || len(refs) == 0 {
		return refs
	}
	sourceBase := resolveNavigationSourceBase(runtime.InvocationCWD)
	normalized := append([]Reference(nil), refs...)
	for i := range normalized {
		normalized[i].File = normalizeReferenceFilePath(normalized[i], targetRoot, sourceBase)
	}
	return normalized
}
