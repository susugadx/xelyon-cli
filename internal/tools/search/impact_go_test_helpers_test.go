package search

import "testing"

func newGoImpactFilePatternSearchOptions(dir string, symbol string, pattern string) SearchOptions {
	return newGoImpactWorkspaceSearchOptions(dir, dir, dir, symbol, pattern)
}

func newGoImpactWorkspaceSearchOptions(root string, cwd string, searchPath string, symbol string, pattern string) SearchOptions {
	return SearchOptions{
		Pattern:            symbol,
		Intent:             "impact",
		Path:               searchPath,
		FilePattern:        pattern,
		ProjectMapRootPath: root,
		InvocationCWD:      cwd,
	}
}

func withWorkingDirForGoImpactTest(t *testing.T, dir string) {
	withWorkingDirForSearchTest(t, dir)
}

func assertGoImpactBundleExcludesEvidenceFile(t *testing.T, bundle *SymbolBundle, file string) {
	t.Helper()
	assertImpactBundleExcludesEvidenceFile(t, bundle, file, "Go")
}
