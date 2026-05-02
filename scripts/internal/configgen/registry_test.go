package configgen

import (
	"slices"
	"testing"
)

func TestFieldTypeToConst(t *testing.T) {
	cases := map[string]string{
		"bool":      "FieldTypeBool",
		"int":       "FieldTypeInt",
		"string":    "FieldTypeString",
		"select":    "FieldTypeSelect",
		"float":     "FieldTypeFloat",
		"[]string":  "FieldTypeStringSlice",
		"map":       "FieldTypeStringMap",
		"structmap": "FieldTypeStructMap",
	}
	for input, want := range cases {
		got, err := FieldTypeToConst(input)
		if err != nil {
			t.Fatalf("FieldTypeToConst(%q) unexpected error: %v", input, err)
		}
		if got != want {
			t.Fatalf("unexpected const for %q: got=%q want=%q", input, got, want)
		}
	}
}

func TestFieldTypeToConstUnknown(t *testing.T) {
	if _, err := FieldTypeToConst("unknown"); err == nil {
		t.Fatal("expected unknown field type to return error")
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

func TestCollectCategoryFields(t *testing.T) {
	got := CollectCategoryFields("provider")
	for _, expected := range []string{"default_model", "default_provider", "provider_models"} {
		if !slices.Contains(got, expected) {
			t.Fatalf("missing expected field %q in %v", expected, got)
		}
	}
}

func TestBuildRegistryEntries(t *testing.T) {
	fieldTypeEntries, err := BuildRegistryFieldTypeEntries()
	if err != nil {
		t.Fatalf("BuildRegistryFieldTypeEntries error: %v", err)
	}
	if len(fieldTypeEntries) == 0 {
		t.Fatal("expected field type entries")
	}
	if !slices.IsSortedFunc(fieldTypeEntries, func(a, b RegistryFieldTypeEntry) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	}) {
		t.Fatal("field type entries should be sorted by path")
	}

	selectEntries := BuildRegistrySelectEntries()
	if len(selectEntries) == 0 {
		t.Fatal("expected select entries")
	}
	if !slices.IsSortedFunc(selectEntries, func(a, b RegistrySelectEntry) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	}) {
		t.Fatal("select entries should be sorted by path")
	}

	descriptionEntries := BuildRegistryDescriptionEntries()
	if len(descriptionEntries) == 0 {
		t.Fatal("expected description entries")
	}
	if !slices.IsSortedFunc(descriptionEntries, func(a, b RegistryDescriptionEntry) int {
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	}) {
		t.Fatal("description entries should be sorted by path")
	}
}
