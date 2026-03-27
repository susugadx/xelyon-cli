package search

import "testing"

func TestSearchCodeToolParameters_RemoveUnusedSearchParams(t *testing.T) {
	params := (&SearchCodeTool{}).Parameters()

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", params["properties"])
	}
	if len(properties) != 5 {
		t.Fatalf("expected 5 parameters after schema update, got %d", len(properties))
	}
	for _, key := range []string{"token_budget", "multiline", "include_hidden", "include_ignored", "context_lines", "file_type", "file_pattern", "output_mode"} {
		if _, exists := properties[key]; exists {
			t.Fatalf("%s should not be exposed in the schema", key)
		}
	}
	if _, exists := properties["file_filter"]; !exists {
		t.Fatal("file_filter should be exposed in the schema")
	}
	if _, exists := properties["mode"]; !exists {
		t.Fatal("mode should be exposed in the schema")
	}
}

func TestSearchCodeToolParameters_ModeSchema(t *testing.T) {
	params := (&SearchCodeTool{}).Parameters()
	properties := params["properties"].(map[string]interface{})
	modeProp := properties["mode"].(map[string]interface{})

	if modeProp["type"] != "string" {
		t.Fatalf("mode.type = %v, want string", modeProp["type"])
	}

	enumVals, ok := modeProp["enum"].([]string)
	if !ok {
		t.Fatalf("mode.enum should be []string, got %T", modeProp["enum"])
	}
	want := []string{"auto", "symbol", "literal", "regex"}
	if len(enumVals) != len(want) {
		t.Fatalf("mode.enum len = %d, want %d", len(enumVals), len(want))
	}
	for i, v := range want {
		if enumVals[i] != v {
			t.Fatalf("mode.enum[%d] = %q, want %q", i, enumVals[i], v)
		}
	}
}

func TestContainsGlobChar(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "go", want: false},
		{input: "*.go", want: true},
		{input: "*_test.go", want: true},
		{input: "Dockerfile*", want: true},
	}

	for _, tt := range tests {
		if got := containsGlobChar(tt.input); got != tt.want {
			t.Fatalf("containsGlobChar(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
