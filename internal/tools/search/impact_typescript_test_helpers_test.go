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

func recommendedReadsContainFile(bundle *SymbolBundle, file string) bool {
	if bundle == nil || bundle.Impact == nil {
		return false
	}
	for _, item := range bundle.Impact.RecommendedReads {
		if item.File == file {
			return true
		}
	}
	return false
}

func bundleSectionsContainFile(bundle *SymbolBundle, file string) bool {
	if bundle == nil {
		return false
	}
	for _, section := range bundle.Sections {
		for _, item := range section.Items {
			if item.File == file {
				return true
			}
		}
	}
	return false
}

func assertTypeScriptImpactBundleExcludesEvidenceFile(t *testing.T, bundle *SymbolBundle, file string) {
	t.Helper()
	if recommendedReadsContainFile(bundle, file) {
		t.Fatalf("%s should not be recommended as impact evidence, got %v", file, recommendedReadFiles(bundle))
	}
	if bundleSectionsContainFile(bundle, file) {
		t.Fatalf("%s should not be emitted in impact sections, got %+v", file, bundle.Sections)
	}
}

func assertRecommendedReadAt(t *testing.T, reads []SymbolBundleItem, index int, kind string, file string) {
	t.Helper()
	if len(reads) <= index {
		t.Fatalf("RecommendedReads length = %d, want index %d", len(reads), index)
	}
	if reads[index].Kind != kind || reads[index].File != file {
		t.Fatalf("RecommendedReads[%d] = (%q, %q), want (%q, %q); all reads: %+v", index, reads[index].Kind, reads[index].File, kind, file, reads)
	}
}

func genericSymbolRefsForTest(prefix string, extension string, count int) []genericSymbolRef {
	refs := make([]genericSymbolRef, 0, count)
	for i := 0; i < count; i++ {
		refs = append(refs, genericSymbolRef{
			File:    prefix + string(rune('a'+i)) + extension,
			Line:    i + 1,
			Snippet: "buildUser()",
		})
	}
	return refs
}
