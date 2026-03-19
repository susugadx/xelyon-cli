package search

import "testing"

func TestSearchCodeToolParameters_RemoveUnusedSearchParams(t *testing.T) {
	params := (&SearchCodeTool{}).Parameters()

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", params["properties"])
	}
	if len(properties) != 4 {
		t.Fatalf("expected 4 parameters after removing search_code params, got %d", len(properties))
	}
	for _, key := range []string{"token_budget", "multiline", "include_hidden", "include_ignored", "context_lines", "file_type", "file_pattern", "output_mode"} {
		if _, exists := properties[key]; exists {
			t.Fatalf("%s should not be exposed in the schema", key)
		}
	}
	if _, exists := properties["file_filter"]; !exists {
		t.Fatal("file_filter should be exposed in the schema")
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
