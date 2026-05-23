package search

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/filefilter"
)

func TestSearchCodeToolParameters_RemoveUnusedSearchParams(t *testing.T) {
	params := (&SearchCodeTool{}).Parameters()

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected properties map, got %T", params["properties"])
	}
	if len(properties) != 6 {
		t.Fatalf("expected 6 parameters after schema update, got %d", len(properties))
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
	if _, exists := properties["intent"]; !exists {
		t.Fatal("intent should be exposed in the schema")
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

func TestFileFilterMatches(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		filter string
		want   bool
	}{
		{name: "empty filter matches", path: "pkg/target.go", filter: "", want: true},
		{name: "language filter matches path", path: "pkg/target.go", filter: "go", want: true},
		{name: "language filter rejects other ext", path: "pkg/target.py", filter: "go", want: false},
		{name: "python filter matches py extension", path: "pkg/models.py", filter: "py", want: true},
		{name: "python filter matches pyi extension", path: "pkg/types.pyi", filter: "py", want: true},
		{name: "python alias matches pyi extension", path: "pkg/types.pyi", filter: "python", want: true},
		{name: "plain rs token matches rs extension", path: "pkg/main.rs", filter: "rs", want: true},
		{name: "plain rs token rejects go extension", path: "pkg/main.go", filter: "rs", want: false},
		{name: "plain json token matches json extension", path: "pkg/config.json", filter: "json", want: true},
		{name: "plain ex token matches ex extension", path: "pkg/main.ex", filter: "ex", want: true},
		{name: "plain exs token matches exs extension", path: "pkg/main.exs", filter: "exs", want: true},
		{name: "c filter matches header extension", path: "pkg/target.h", filter: "c", want: true},
		{name: "c filter matches generated header template", path: "pkg/target.h.in", filter: "c", want: true},
		{name: "java filter matches properties extension", path: "pkg/application.properties", filter: "java", want: true},
		{name: "java filter matches jspx extension", path: "pkg/view.jspx", filter: "java", want: true},
		{name: "sh filter matches zsh extension", path: "pkg/env.zsh", filter: "sh", want: true},
		{name: "sh filter matches shell dotfile", path: "pkg/.bashrc", filter: "sh", want: true},
		{name: "typescript alias matches ts extension", path: "pkg/App.ts", filter: "typescript", want: true},
		{name: "typescript alias matches tsx extension", path: "pkg/App.tsx", filter: "typescript", want: true},
		{name: "typescript alias rejects cts extension", path: "pkg/App.cts", filter: "typescript", want: false},
		{name: "typescript alias rejects mts extension", path: "pkg/App.mts", filter: "typescript", want: false},
		{name: "typescript alias rejects go extension", path: "pkg/App.go", filter: "typescript", want: false},
		{name: "javascript alias matches js extension", path: "pkg/App.js", filter: "javascript", want: true},
		{name: "javascript alias matches jsx extension", path: "pkg/App.jsx", filter: "javascript", want: true},
		{name: "javascript alias matches mjs extension", path: "pkg/App.mjs", filter: "javascript", want: true},
		{name: "javascript alias matches cjs extension", path: "pkg/App.cjs", filter: "javascript", want: true},
		{name: "javascript alias rejects vue extension", path: "pkg/App.vue", filter: "javascript", want: false},
		{name: "glob matches basename", path: "pkg/target_test.go", filter: "*_test.go", want: true},
		{name: "glob matches clean path", path: "pkg/generated/mock.go", filter: "pkg/generated/*.go", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filefilter.Matches(tt.path, tt.filter); got != tt.want {
				t.Fatalf("filefilter.Matches(%q, %q) = %v, want %v", tt.path, tt.filter, got, tt.want)
			}
		})
	}
}

func TestSearchCodeToolParameters_FileFilterDescriptionMatchesContract(t *testing.T) {
	params := (&SearchCodeTool{}).Parameters()
	properties := params["properties"].(map[string]interface{})
	fileFilterProp := properties["file_filter"].(map[string]interface{})
	description, ok := fileFilterProp["description"].(string)
	if !ok {
		t.Fatalf("file_filter.description should be string, got %T", fileFilterProp["description"])
	}
	if !strings.Contains(description, "ripgrep-like file type mapping") {
		t.Fatalf("file_filter description should mention ripgrep-like mapping, got %q", description)
	}
	if strings.Contains(description, "uses rg --type") {
		t.Fatalf("file_filter description should avoid false rg --type precision, got %q", description)
	}
}

func TestFileFilterParse(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		wantType    string
		wantPattern string
	}{
		{name: "trims and lowercases extension token", filter: "  RS  ", wantType: "rs"},
		{name: "strips leading dot token", filter: " .json ", wantType: "json"},
		{name: "keeps glob pattern", filter: "  *_test.go  ", wantPattern: "*_test.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotPattern := filefilter.Parse(tt.filter)
			if gotType != tt.wantType || gotPattern != tt.wantPattern {
				t.Fatalf("filefilter.Parse(%q) = (%q, %q), want (%q, %q)", tt.filter, gotType, gotPattern, tt.wantType, tt.wantPattern)
			}
		})
	}
}

func TestSearchCodeToolParameters_ImpactDescriptionMatchesStructuredLanguageBehavior(t *testing.T) {
	params := (&SearchCodeTool{}).Parameters()
	properties := params["properties"].(map[string]interface{})
	intentProp := properties["intent"].(map[string]interface{})
	description, ok := intentProp["description"].(string)
	if !ok {
		t.Fatalf("intent.description should be string, got %T", intentProp["description"])
	}

	for _, want := range []string{
		"structured Go, TypeScript .ts, targeted TSX .tsx, or JavaScript .js/.jsx single-symbol impact path",
		"file_filter=go",
		"*.go",
		"scoped Go **/*.go globs",
		"direct .go paths",
		"file_filter=tsx",
		"*.tsx",
		"direct .tsx paths",
		"file_filter=js",
		"*.js",
		"direct .js paths",
		"file_filter=jsx",
		"*.jsx",
		"direct .jsx paths",
		"file_filter=typescript and file_filter=javascript remain broad fallback scopes",
		".mjs/.cjs are not JavaScript structured impact targets",
		"conservative related multi-pattern search",
	} {
		if !strings.Contains(description, want) {
			t.Fatalf("expected intent description to mention %q, got %q", want, description)
		}
	}
}
