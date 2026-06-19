package configmeta

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func TestOrderedSectionsForCategory(t *testing.T) {
	got := OrderedSectionsForCategory("provider")
	want := []string{"default_provider", "default_model", "provider_models", "gemini"}
	if len(got) != len(want) {
		t.Fatalf("unexpected section count: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected section order: got=%v want=%v", got, want)
		}
	}
}

func TestCanonicalFieldPath(t *testing.T) {
	if got := CanonicalFieldPath("general", "ui_language"); got != "general.ui_language" {
		t.Fatalf("unexpected canonical path: %s", got)
	}
	if got := CanonicalFieldPath("default_provider", "default_provider"); got != "default_provider" {
		t.Fatalf("unexpected top-level canonical path: %s", got)
	}
}

func TestSectionOrderHasCategoryMapping(t *testing.T) {
	for _, sectionName := range SectionOrder {
		if _, ok := SectionToCategory[sectionName]; !ok {
			t.Fatalf("missing category mapping for %q", sectionName)
		}
		if _, ok := Sections[sectionName]; !ok {
			t.Fatalf("missing section metadata for %q", sectionName)
		}
	}
}

func TestCategoryOrderHasCategoryDefinition(t *testing.T) {
	for _, categoryName := range CategoryOrder {
		if _, ok := Categories[categoryName]; !ok {
			t.Fatalf("missing category definition for %q", categoryName)
		}
		sections := OrderedSectionsForCategory(categoryName)
		if len(sections) == 0 {
			t.Fatalf("category %q has no sections", categoryName)
		}
	}
}

func TestUserFacingSectionsHaveStructNames(t *testing.T) {
	skip := map[string]struct{}{
		"default_provider": {},
		"default_model":    {},
		"provider_models":  {},
	}
	for _, sectionName := range SectionOrder {
		if _, ok := skip[sectionName]; ok {
			continue
		}
		if Sections[sectionName].StructName == "" {
			t.Fatalf("expected StructName for %q", sectionName)
		}
	}
}

func TestSectionMetadataFieldConsistency(t *testing.T) {
	for sectionName, section := range Sections {
		for fieldName := range section.Fields {
			if _, ok := section.FieldTypes[fieldName]; !ok {
				t.Fatalf("missing FieldTypes for %s.%s", sectionName, fieldName)
			}
		}
		for fieldName, fieldType := range section.FieldTypes {
			if fieldType == "select" {
				if _, ok := section.SelectOpts[fieldName]; !ok {
					t.Fatalf("missing SelectOpts for select field %s.%s", sectionName, fieldName)
				}
			}
		}
		for fieldName := range section.SelectOpts {
			if section.FieldTypes[fieldName] != "select" {
				t.Fatalf("SelectOpts defined for non-select field %s.%s", sectionName, fieldName)
			}
		}

		switch section.Example.FilterMode {
		case "", ExampleFilterModeFields, ExampleFilterModeKeepAll:
		default:
			t.Fatalf("unknown example filter mode %q for %s", section.Example.FilterMode, sectionName)
		}
	}
}

func TestAgentInstructionsProjectFilesDescriptionDocumentsScopedGuidance(t *testing.T) {
	description := Sections["agent_instructions"].Fields["project.files"]
	for _, expected := range []string{
		"basename",
		"root→cwd",
		"root→入力参照 path",
		"scoped chain",
		"root 相対 explicit file",
	} {
		if !strings.Contains(description, expected) {
			t.Fatalf("project.files description missing %q: %s", expected, description)
		}
	}
}

func TestProviderSelectOptionsUseLLMCatalog(t *testing.T) {
	if got, want := Sections["default_provider"].SelectOpts["default_provider"], llmcatalog.DisplayProviderKeys(); !reflect.DeepEqual(got, want) {
		t.Fatalf("default_provider options = %v, want %v", got, want)
	}
	if got, want := Sections["web_search"].SelectOpts["provider"], llmcatalog.NativeWebSearchProviderKeys(true); !reflect.DeepEqual(got, want) {
		t.Fatalf("web_search.provider options = %v, want %v", got, want)
	}
}

func TestExamplePolicyDefaults(t *testing.T) {
	providerModels := Sections["provider_models"]
	if providerModels.Example.FilterMode != ExampleFilterModeKeepAll {
		t.Fatalf("provider_models should keep all fields in example, got %q", providerModels.Example.FilterMode)
	}
	lsp := Sections["lsp"]
	if !lsp.Example.OmittedFields["servers"] {
		t.Fatal("lsp.servers should be omitted from config example")
	}
	if _, ok := lsp.Example.Overrides["servers"]; !ok {
		t.Fatal("lsp.servers override should exist in metadata")
	}
	webSearch := Sections["web_search"]
	if got := webSearch.Example.Overrides["provider"]; got != "gemini" {
		t.Fatalf("web_search provider override = %#v, want gemini", got)
	}
}
