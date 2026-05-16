package search

import "testing"

func newTypeScriptImpactSearchOptions(dir string, symbol string) SearchOptions {
	return SearchOptions{
		Pattern:       symbol,
		Intent:        "impact",
		Path:          dir,
		FileType:      "ts",
		InvocationCWD: dir,
	}
}

func newTypeScriptImpactFilePatternSearchOptions(dir string, symbol string, pattern string) SearchOptions {
	opts := newTypeScriptImpactSearchOptions(dir, symbol)
	opts.FileType = ""
	opts.FilePattern = pattern
	return opts
}

func newTSXImpactSearchOptions(dir string, symbol string) SearchOptions {
	opts := newTypeScriptImpactSearchOptions(dir, symbol)
	opts.FileType = "tsx"
	return opts
}

func assertTypeScriptStructuredImpactArtifact(t *testing.T, artifact SearchExecutionArtifact, symbol string, kind string) {
	t.Helper()
	assertJSFamilyStructuredImpactArtifact(t, artifact, "typescript", symbol, kind)
}

func assertTypeScriptImpactBundleExcludesEvidenceFile(t *testing.T, bundle *SymbolBundle, file string) {
	t.Helper()
	assertImpactBundleExcludesEvidenceFile(t, bundle, file, "TypeScript")
}
