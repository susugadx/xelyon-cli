package search

import "testing"

func recommendedReadFiles(bundle *SymbolBundle) []string {
	if bundle == nil || bundle.Impact == nil {
		return nil
	}
	files := make([]string, 0, len(bundle.Impact.RecommendedReads))
	for _, item := range bundle.Impact.RecommendedReads {
		files = append(files, item.File)
	}
	return files
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

func assertImpactBundleExcludesEvidenceFile(t *testing.T, bundle *SymbolBundle, file string, label string) {
	t.Helper()
	if recommendedReadsContainFile(bundle, file) {
		t.Fatalf("%s should not be recommended as %s impact evidence, got %v", file, label, recommendedReadFiles(bundle))
	}
	if bundleSectionsContainFile(bundle, file) {
		t.Fatalf("%s should not be emitted in %s impact sections, got %+v", file, label, bundle.Sections)
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
