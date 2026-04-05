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
		"unknown":   "FieldTypeString",
	}
	for input, want := range cases {
		if got := FieldTypeToConst(input); got != want {
			t.Fatalf("unexpected const for %q: got=%q want=%q", input, got, want)
		}
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
