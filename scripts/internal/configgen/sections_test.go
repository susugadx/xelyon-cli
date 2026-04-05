package configgen

import "testing"

func TestOrderedSectionsForCategory(t *testing.T) {
	got := OrderedSectionsForCategory("provider")
	want := []string{"default_provider", "default_model", "provider_models"}
	if len(got) != len(want) {
		t.Fatalf("unexpected section count: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected section order: got=%v want=%v", got, want)
		}
	}
}

func TestSectionOrderHasCategoryMapping(t *testing.T) {
	for _, sectionName := range SectionOrder {
		if _, ok := SectionToCategory[sectionName]; !ok {
			t.Fatalf("missing category mapping for %q", sectionName)
		}
	}
}

func TestUserFacingSectionsHaveStructNames(t *testing.T) {
	for _, sectionName := range []string{
		"general",
		"compression",
		"execution",
		"paste",
		"project_map",
		"lsp",
		"output",
		"web_search",
		"sub_agent",
		"mcp",
		"hooks",
	} {
		if Sections[sectionName].StructName == "" {
			t.Fatalf("expected StructName for %q", sectionName)
		}
	}
}
