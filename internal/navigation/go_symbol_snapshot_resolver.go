package navigation

func resolveSymbolCandidatesWithRuntime(symbol, pathHint string, runtime GoSymbolRuntime) []SymbolCandidate {
	query := parseSymbolQuery(symbol)
	if candidates := resolveSymbolCandidatesFromSnapshotSource(query, pathHint, runtime); len(candidates) > 0 {
		return candidates
	}
	return resolveSymbolCandidatesFromASTSource(query, pathHint, runtime)
}

func findAmbiguousFilesWithRuntime(symbol string, cand SymbolCandidate, runtime GoSymbolRuntime) map[string]bool {
	ambiguous := findAmbiguousFilesFromSnapshot(symbol, cand, runtime)
	if len(ambiguous) > 0 {
		return ambiguous
	}

	allCandidates := resolveSymbolCandidatesWithRuntime(symbol, "", runtime)
	for _, candidate := range allCandidates {
		if candidate.File == "" || candidate.File == cand.File {
			continue
		}
		ambiguous[candidate.File] = true
	}
	return ambiguous
}
