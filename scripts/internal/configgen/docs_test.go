package configgen

import (
	"go/ast"
	"go/parser"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReplaceMarkerContent(t *testing.T) {
	input := "before\n<!-- START -->\nold\n<!-- END -->\nafter\n"
	got, ok := ReplaceMarkerContent(input, "<!-- START -->", "<!-- END -->", "new")
	if !ok {
		t.Fatal("expected marker replacement to succeed")
	}
	if !strings.Contains(got, "<!-- START -->\nnew\n<!-- END -->") {
		t.Fatalf("unexpected replacement result: %s", got)
	}
}

func TestReplaceMarkerContentMissingMarker(t *testing.T) {
	got, ok := ReplaceMarkerContent("no markers", "<!-- START -->", "<!-- END -->", "new")
	if ok {
		t.Fatal("expected missing marker replacement to fail")
	}
	if got != "no markers" {
		t.Fatalf("unexpected content mutation: %s", got)
	}
}

func TestParseConfigTypesParsesAllFilesInDirectory(t *testing.T) {
	dir := t.TempDir()
	source1 := `package config

// FirstConfig comment
type FirstConfig struct {
	// First field comment
	Name string ` + "`yaml:\"name,omitempty\"`" + `
}
`
	source2 := `package config

// SecondConfig comment
type SecondConfig struct {
	Enabled bool ` + "`yaml:\"enabled\"`" + `
	Nested *ThirdConfig ` + "`yaml:\"nested,omitempty\"`" + `
}

type ThirdConfig struct {
	Value string ` + "`yaml:\"value\"`" + `
}
`
	testSource := `package config
type IgnoredConfig struct {
	Value string ` + "`yaml:\"value\"`" + `
}
`

	if err := os.WriteFile(filepath.Join(dir, "first.go"), []byte(source1), 0o644); err != nil {
		t.Fatalf("write first.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "second.go"), []byte(source2), 0o644); err != nil {
		t.Fatalf("write second.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ignored_test.go"), []byte(testSource), 0o644); err != nil {
		t.Fatalf("write ignored_test.go: %v", err)
	}

	structs, err := ParseConfigTypes(dir)
	if err != nil {
		t.Fatalf("ParseConfigTypes returned error: %v", err)
	}
	if len(structs) != 3 {
		t.Fatalf("unexpected struct count: %#v", structs)
	}
	if structs[0].Name != "FirstConfig" || structs[1].Name != "SecondConfig" || structs[2].Name != "ThirdConfig" {
		t.Fatalf("unexpected parsed structs: %#v", structs)
	}
	if structs[0].Fields[0].YAMLTag != "name" || !structs[0].Fields[0].IsOptional {
		t.Fatalf("unexpected first field parsing: %#v", structs[0].Fields[0])
	}
	if structs[1].Fields[1].Type != "*ThirdConfig" {
		t.Fatalf("expected pointer type, got %#v", structs[1].Fields[1])
	}
}

func TestParseConfigTypesUsesTrailingFieldComment(t *testing.T) {
	dir := t.TempDir()
	source := `package config

type CommentConfig struct {
	Name string ` + "`yaml:\"name\"`" + ` // trailing comment
}
`
	if err := os.WriteFile(filepath.Join(dir, "comment.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write comment.go: %v", err)
	}

	structs, err := ParseConfigTypes(dir)
	if err != nil {
		t.Fatalf("ParseConfigTypes returned error: %v", err)
	}
	if len(structs) != 1 || len(structs[0].Fields) != 1 {
		t.Fatalf("unexpected parsed structs: %#v", structs)
	}
	if structs[0].Fields[0].Comment != "trailing comment" {
		t.Fatalf("unexpected trailing field comment: %#v", structs[0].Fields[0])
	}
}

func TestParseConfigTypesRejectsUnsupportedTypeExpression(t *testing.T) {
	dir := t.TempDir()
	source := `package config

type InvalidConfig struct {
	Value <-chan string ` + "`yaml:\"value\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "invalid.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write invalid.go: %v", err)
	}
	if _, err := ParseConfigTypes(dir); err == nil {
		t.Fatal("expected ParseConfigTypes to fail for unsupported type")
	}
}

func TestParseConfigTypesExpandsMultiFieldNames(t *testing.T) {
	dir := t.TempDir()
	source := `package config

type MultiFieldConfig struct {
	Primary, Secondary string ` + "`yaml:\"name,omitempty\"`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "multi.go"), []byte(source), 0o644); err != nil {
		t.Fatalf("write multi.go: %v", err)
	}

	structs, err := ParseConfigTypes(dir)
	if err != nil {
		t.Fatalf("ParseConfigTypes returned error: %v", err)
	}
	if len(structs) != 1 {
		t.Fatalf("unexpected struct count: %#v", structs)
	}
	if len(structs[0].Fields) != 2 {
		t.Fatalf("unexpected field count: %#v", structs[0].Fields)
	}
	if structs[0].Fields[0].Name != "Primary" || structs[0].Fields[1].Name != "Secondary" {
		t.Fatalf("unexpected field names: %#v", structs[0].Fields)
	}
}

func TestGenerateConfigDetails(t *testing.T) {
	structs := []StructInfo{
		{
			Name:    "GeneralConfig",
			Comment: "General config comment\nSecond line",
			Fields: []FieldInfo{
				{Name: "UILanguage", Type: "string", YAMLTag: "ui_language", Comment: "表示言語"},
				{Name: "Hidden", Type: "string", YAMLTag: "hidden", Comment: "内部: omit"},
			},
		},
	}
	defaults := map[string]interface{}{
		"general": map[string]interface{}{
			"ui_language": "auto",
			"hidden":      "secret",
		},
	}

	output := GenerateConfigDetails(structs, defaults)
	for _, expected := range []string{
		"### 一般設定 (`general`)",
		"General config comment",
		"ui_language: auto",
		"#### `ui_language`",
		"- **デフォルト**: `auto`",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected details to contain %q, got %s", expected, output)
		}
	}
	if strings.Contains(output, "hidden") {
		t.Fatalf("internal field should not appear in details: %s", output)
	}
}

func TestGenerateConfigDetailsHandlesMapAndUnknownDefaults(t *testing.T) {
	structs := []StructInfo{
		{
			Name:    "WebSearchConfig",
			Comment: "Web search settings",
			Fields: []FieldInfo{
				{Name: "Provider", Type: "string", YAMLTag: "provider", Comment: "Provider description (デフォルト: openai)"},
				{Name: "Headers", Type: "map[string]string", YAMLTag: "headers", Comment: "Header map"},
				{Name: "Skip", Type: "string", YAMLTag: "-", Comment: "ignored"},
				{Name: "Missing", Type: "string", YAMLTag: "", Comment: "ignored too"},
			},
		},
	}
	defaults := map[string]interface{}{
		"web_search": map[string]interface{}{
			"provider": "gemini",
		},
	}

	output := GenerateConfigDetails(structs, defaults)
	for _, expected := range []string{
		"### Web検索設定 (`web_search`)",
		"headers: { ... }",
		"- **説明**: Provider description",
	} {
		if !strings.Contains(output, expected) {
			t.Fatalf("expected details to contain %q, got %s", expected, output)
		}
	}
	for _, unexpected := range []string{"#### `headers`\n- **デフォルト**", "#### `-`", "#### ``"} {
		if strings.Contains(output, unexpected) {
			t.Fatalf("unexpected details content %q in %s", unexpected, output)
		}
	}
}

func TestFormatConfigExample(t *testing.T) {
	input := configExampleFileHeader + "# ============================================================\n# 一般設定\n# ============================================================\ngeneral:\n  ui_language: auto\n"
	got := FormatConfigExample(input)
	if strings.Contains(got, "# XELYON CLI 設定例") {
		t.Fatalf("expected file header comments to be removed: %s", got)
	}
	for _, expected := range []string{
		"```yaml",
		"# ============================================================",
		"# 一般設定",
		"general:",
		"ui_language: auto",
		"```",
	} {
		if !strings.Contains(got, expected) {
			t.Fatalf("unexpected formatted example, missing %q in %s", expected, got)
		}
	}
}

func TestParseConfigTypesError(t *testing.T) {
	if _, err := ParseConfigTypes(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected ParseConfigTypes to return an error for missing dir")
	}
}

func TestDocsHelperFunctions(t *testing.T) {
	typeExpr := func(src string) ast.Expr {
		t.Helper()
		expr, err := parser.ParseExpr(src)
		if err != nil {
			t.Fatalf("ParseExpr(%q): %v", src, err)
		}
		return expr
	}

	if got, err := getTypeString(typeExpr("[]string")); err != nil || got != "[]string" {
		t.Fatalf("unexpected array type: got=%q err=%v", got, err)
	}
	if got, err := getTypeString(typeExpr("map[string]int")); err != nil || got != "map[string]int" {
		t.Fatalf("unexpected map type: got=%q err=%v", got, err)
	}
	if got, err := getTypeString(typeExpr("pkg.Type")); err != nil || got != "pkg.Type" {
		t.Fatalf("unexpected selector type: got=%q err=%v", got, err)
	}
	if got, err := getTypeString(typeExpr("*Thing")); err != nil || got != "*Thing" {
		t.Fatalf("unexpected pointer type: got=%q err=%v", got, err)
	}
	if _, err := getTypeString(&ast.ChanType{Value: ast.NewIdent("Thing")}); err == nil {
		t.Fatal("expected channel type to be unsupported")
	}

	defaults := map[string]interface{}{"general": map[string]interface{}{"ui_language": "auto"}}
	if got := getDefaultValue(defaults, "general", "ui_language"); got != "auto" {
		t.Fatalf("unexpected default value: %#v", got)
	}
	if got := getDefaultValue(defaults, "general", "missing"); got != nil {
		t.Fatalf("expected missing default to be nil, got %#v", got)
	}
	if got := getDefaultValue(defaults, "missing", "field"); got != nil {
		t.Fatalf("expected missing section default to be nil, got %#v", got)
	}

	yamlValueCases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil", nil, "null"},
		{"bool", true, "true"},
		{"number", 42, "42"},
		{"string", "abc", "abc"},
		{"empty-string", "", `""`},
		{"empty-slice", []interface{}{}, "[]"},
		{"slice", []interface{}{"a"}, "[...]"},
		{"other", map[string]interface{}{"a": 1}, "..."},
	}
	for _, tc := range yamlValueCases {
		if got := formatYAMLValue(tc.in); got != tc.want {
			t.Fatalf("formatYAMLValue(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}

	defaultValueCases := []struct {
		name string
		in   interface{}
		want string
	}{
		{"nil", nil, "null"},
		{"bool", false, "false"},
		{"number", 7, "7"},
		{"string", "abc", "abc"},
		{"empty-string", "", `""`},
		{"empty-slice", []interface{}{}, "[]"},
		{"slice", []interface{}{"a", 2}, "[a, 2]"},
		{"string-slice", []string{"a", "b"}, "[a, b]"},
		{"other", map[string]int{"a": 1}, "map[a:1]"},
	}
	for _, tc := range defaultValueCases {
		if got := formatDefaultValue(tc.in); got != tc.want {
			t.Fatalf("formatDefaultValue(%s) = %q, want %q", tc.name, got, tc.want)
		}
	}

	displayCases := map[string]string{
		"bool":              "boolean",
		"int":               "integer",
		"int64":             "integer",
		"string":            "string",
		"[]string":          "string[]",
		"map[string]string": "map",
		"custom.Type":       "custom.Type",
	}
	for input, want := range displayCases {
		if got := mapGoTypeToDisplay(input); got != want {
			t.Fatalf("mapGoTypeToDisplay(%q) = %q, want %q", input, got, want)
		}
	}

	if got := extractDescription(""); got != "" {
		t.Fatalf("expected empty description, got %q", got)
	}
	if got := extractDescription("説明文 (デフォルト: 123)"); got != "説明文" {
		t.Fatalf("unexpected extracted description: %q", got)
	}
}
