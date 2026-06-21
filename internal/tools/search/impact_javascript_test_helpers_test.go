package search

import "testing"

func newJavaScriptImpactSearchOptions(dir string, symbol string) SearchOptions {
	return SearchOptions{
		Pattern:       symbol,
		Intent:        "impact",
		Path:          dir,
		FileType:      "js",
		InvocationCWD: dir,
	}
}

func newJavaScriptImpactFilePatternSearchOptions(dir string, symbol string, pattern string) SearchOptions {
	opts := newJavaScriptImpactSearchOptions(dir, symbol)
	opts.FileType = ""
	opts.FilePattern = pattern
	return opts
}

func assertJavaScriptStructuredImpactArtifact(t *testing.T, artifact SearchExecutionArtifact, symbol string, kind string) {
	t.Helper()
	assertJSFamilyStructuredImpactArtifact(t, artifact, "javascript", symbol, kind)
}
