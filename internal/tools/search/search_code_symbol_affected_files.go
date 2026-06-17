package search

func resolveMultiSymbolAffectedFiles(resolved symbolResolveResult, opts SearchOptions) []string {
	affectedFiles := append([]string(nil), resolved.AffectedFiles...)
	affectedFiles = append(affectedFiles, collectPrimaryAffectedFilePathsFromOutput(resolved.Output, opts)...)
	affectedFiles = dedupePaths(affectedFiles)
	if len(affectedFiles) > 0 {
		return affectedFiles
	}
	return deriveAffectedFilesFromCachedResult(nil, resolved.Output, opts)
}
