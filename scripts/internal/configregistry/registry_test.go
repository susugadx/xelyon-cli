package configregistry

import (
	"slices"
	"strings"
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
		got, err := fieldTypeToConst(input)
		if err != nil {
			t.Fatalf("FieldTypeToConst(%q) unexpected error: %v", input, err)
		}
		if got != want {
			t.Fatalf("unexpected const for %q: got=%q want=%q", input, got, want)
		}
	}
}

func TestFieldTypeToConstUnknown(t *testing.T) {
	if _, err := fieldTypeToConst("unknown"); err == nil {
		t.Fatal("expected unknown field type to return error")
	}
}

func TestCollectCategoryFields(t *testing.T) {
	got := collectCategoryFields("provider")
	for _, expected := range []string{"default_model", "default_provider", "provider_models"} {
		if !slices.Contains(got, expected) {
			t.Fatalf("missing expected field %q in %v", expected, got)
		}
	}
}

func TestBuildRegistryEntries(t *testing.T) {
	fieldTypeEntries, err := buildRegistryFieldTypeEntries()
	if err != nil {
		t.Fatalf("BuildRegistryFieldTypeEntries error: %v", err)
	}
	if len(fieldTypeEntries) == 0 {
		t.Fatal("expected field type entries")
	}
	if !slices.IsSortedFunc(fieldTypeEntries, func(a, b registryFieldTypeEntry) int {
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

	selectEntries := buildRegistrySelectEntries()
	if len(selectEntries) == 0 {
		t.Fatal("expected select entries")
	}
	if !slices.IsSortedFunc(selectEntries, func(a, b registrySelectEntry) int {
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

	descriptionEntries := buildRegistryDescriptionEntries()
	if len(descriptionEntries) == 0 {
		t.Fatal("expected description entries")
	}
	if !slices.IsSortedFunc(descriptionEntries, func(a, b registryDescriptionEntry) int {
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

func TestGenerateRegistrySourceIncludesMCPMetadata(t *testing.T) {
	source, err := GenerateRegistrySource()
	if err != nil {
		t.Fatalf("GenerateRegistrySource error: %v", err)
	}
	text := string(source)
	for _, expected := range []string{
		`{Name: "mcp", DisplayName: "MCP Servers", Icon: "🔌", Fields: []string{"mcp.enabled", "mcp.headless"}}`,
		`"mcp.enabled":`,
		`"mcp.headless":`,
		`FieldTypeBool`,
		`"MCP接続を有効化（デフォルト: true）"`,
		`"Headlessモードでも接続（デフォルト: false）"`,
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("generated registry source missing %q in:\n%s", expected, text)
		}
	}
}

func TestGenerateRegistrySourceIncludesProjectFilesScopedGuidanceDescription(t *testing.T) {
	source, err := GenerateRegistrySource()
	if err != nil {
		t.Fatalf("GenerateRegistrySource error: %v", err)
	}
	text := string(source)
	for _, expected := range []string{
		`"agent_instructions.project.files":`,
		"basename は root→cwd / root→入力参照 path の scoped chain",
		"/ を含む path は root 相対 explicit file",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("generated registry source missing %q in:\n%s", expected, text)
		}
	}
}
