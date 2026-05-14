package search

import (
	"strings"
	"testing"
)

func assertJSFamilyStructuredImpactArtifact(t *testing.T, artifact SearchExecutionArtifact, language string, symbol string, kind string) {
	t.Helper()
	if !artifact.Metadata.StructuredImpact {
		t.Fatalf("StructuredImpact = false, want true; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Ambiguous {
		t.Fatalf("Ambiguous = true, want false; output:\n%s", artifact.Rendered)
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatalf("Bundle = nil, want %s structured bundle; output:\n%s", language, artifact.Rendered)
	}
	if artifact.Metadata.Bundle.Identity.Language != language {
		t.Fatalf("bundle language = %q, want %s", artifact.Metadata.Bundle.Identity.Language, language)
	}
	if artifact.Metadata.Bundle.Identity.DisplayName != symbol {
		t.Fatalf("bundle display name = %q, want %q", artifact.Metadata.Bundle.Identity.DisplayName, symbol)
	}
	if artifact.Metadata.Bundle.Identity.Kind != kind {
		t.Fatalf("bundle kind = %q, want %q", artifact.Metadata.Bundle.Identity.Kind, kind)
	}
	if artifact.Metadata.Bundle.Impact == nil {
		t.Fatalf("Bundle.Impact = nil, want %s impact metadata", language)
	}
	if len(artifact.Metadata.Bundle.Impact.RecommendedReads) == 0 {
		t.Fatal("RecommendedReads is empty, want definition read")
	}
}

func symbolBundleSectionItems(bundle *SymbolBundle, kind string) []SymbolBundleItem {
	if bundle == nil {
		return nil
	}
	for _, section := range bundle.Sections {
		if section.Kind == kind {
			return section.Items
		}
	}
	return nil
}

func symbolBundleItemsContainSnippet(items []SymbolBundleItem, snippet string) bool {
	for _, item := range items {
		if strings.Contains(item.Snippet, snippet) {
			return true
		}
	}
	return false
}

func symbolBundleItemsContainFile(items []SymbolBundleItem, file string) bool {
	for _, item := range items {
		if item.File == file {
			return true
		}
	}
	return false
}

func assertJSFamilyImpactSectionContainsFile(t *testing.T, artifact SearchExecutionArtifact, sectionKind string, file string) {
	t.Helper()
	items := symbolBundleSectionItems(artifact.Metadata.Bundle, sectionKind)
	if !symbolBundleItemsContainFile(items, file) {
		t.Fatalf("%s section = %+v, want %s", sectionKind, items, file)
	}
}

func testLSPRangeForSearchToken(line, token string) (int, int) {
	idx := strings.Index(line, token)
	if idx < 0 {
		panic("token not found: " + token)
	}
	start := testLSPCharacterWidthForSearch(line[:idx]) + 1
	return start, start + testLSPCharacterWidthForSearch(token)
}

func testLSPCharacterWidthForSearch(text string) int {
	width := 0
	for _, r := range text {
		if r > 0xffff {
			width += 2
		} else {
			width++
		}
	}
	return width
}
