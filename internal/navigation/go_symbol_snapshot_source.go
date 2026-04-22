package navigation

import "strings"

func resolveSymbolCandidatesFromSnapshotSource(query symbolQuery, pathHint string, runtime GoSymbolRuntime) []SymbolCandidate {
	snapshot := loadGoSymbolSnapshot(runtime)
	if snapshot == nil || query.BaseName == "" {
		return nil
	}

	matchesPath := buildSnapshotPathMatcher(snapshot.RootPath, runtime.InvocationCWD, pathHint)
	entries := snapshot.ByName[query.BaseName]
	candidates := make([]SymbolCandidate, 0, len(entries))
	for _, entry := range entries {
		if query.Receiver != "" {
			if entry.Kind != "method" || entry.ReceiverNorm != query.Receiver {
				continue
			}
		}
		if matchesPath != nil && !matchesPath(entry.File) {
			continue
		}
		candidates = append(candidates, newSymbolCandidateFromSnapshotEntry(entry, snapshot.RootPath))
	}

	sortSymbolCandidates(candidates)
	return candidates
}

func findAmbiguousFilesFromSnapshot(symbol string, cand SymbolCandidate, runtime GoSymbolRuntime) map[string]bool {
	ambiguous := make(map[string]bool)
	snapshot := loadGoSymbolSnapshot(runtime)
	if snapshot == nil {
		return ambiguous
	}

	for _, entry := range snapshot.ByName[strings.TrimSpace(symbol)] {
		if entry.File == "" || entry.File == cand.File {
			continue
		}
		ambiguous[entry.File] = true
	}
	return ambiguous
}
