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
	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want true; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = true, want false; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want TypeScript structured bundle; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle.Identity.Language != "typescript" {
		t.Fatalf("bundle language = %q, want typescript", artifact.Metadata.Bundle.Identity.Language)
	}
	if artifact.Metadata.Bundle.Identity.DisplayName != symbol {
		t.Fatalf("bundle display name = %q, want %q", artifact.Metadata.Bundle.Identity.DisplayName, symbol)
	}
	if artifact.Metadata.Bundle.Identity.Kind != kind {
		t.Fatalf("bundle kind = %q, want %q", artifact.Metadata.Bundle.Identity.Kind, kind)
	}
	if artifact.Metadata.Bundle.Impact == nil {
		t.Fatal("Bundle.Impact = nil, want TypeScript impact metadata")
	}
	if len(artifact.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("RecommendedReads is empty, want definition read")
	}
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

func assertRecommendedReadAt(t *testing.T, reads []SymbolBundleItem, index int, kind string, file string) {
	t.Helper()
	if len(reads) <= index {
		t.Fatalf("RecommendedReads length = %d, want index %d", len(reads), index)
	}
	if reads[index].Kind != kind || reads[index].File != file {
		t.Fatalf("RecommendedReads[%d] = (%q, %q), want (%q, %q); all reads: %+v", index, reads[index].Kind, reads[index].File, kind, file, reads)
	}
}

func typeScriptGenericRefs(prefix string, count int) []genericSymbolRef {
	refs := make([]genericSymbolRef, 0, count)
	for i := 0; i < count; i++ {
		refs = append(refs, genericSymbolRef{
			File:    prefix + string(rune('a'+i)) + ".ts",
			Line:    i + 1,
			Snippet: "buildUser()",
		})
	}
	return refs
}
