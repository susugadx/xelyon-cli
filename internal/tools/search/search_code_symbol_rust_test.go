package search

import (
	"strings"
	"testing"
)

func TestClassifyRustRefs(t *testing.T) {
	refs := []genericSymbolRef{
		{File: "main.rs", Line: 1, Snippet: "use crate::config::Config;"},
		{File: "main.rs", Line: 5, Snippet: "let cfg = Config::new()"},
		{File: "main.rs", Line: 7, Snippet: "impl Config {"},
		{File: "main.rs", Line: 9, Snippet: "impl Display for Config {"},
		{File: "main.rs", Line: 11, Snippet: "// Config is important"},
		{File: "main.rs", Line: 13, Snippet: "Config(value)"},
		{File: "main.rs", Line: 15, Snippet: "dyn Config"},
		{File: "main.rs", Line: 17, Snippet: "config_log!(Config)"},
	}

	uses, callers, implRefs, others := classifyRustRefs(refs, "Config")

	if len(uses) != 1 {
		t.Errorf("expected 1 use, got %d: %+v", len(uses), uses)
	}
	if len(callers) != 2 {
		t.Errorf("expected 2 callers (::new + direct call), got %d: %+v", len(callers), callers)
	}
	if len(implRefs) != 3 {
		t.Errorf("expected 3 impl refs (impl + impl for + dyn), got %d: %+v", len(implRefs), implRefs)
	}
	if len(others) != 2 {
		t.Errorf("expected 2 others (comment + macro arg), got %d: %+v", len(others), others)
	}
}

func TestSearchCode_RustSymbolSingleHit(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs":  "pub struct Config {\n    pub name: String,\n}\n",
		"main.rs": "use crate::Config;\nlet c = Config { name: \"x\".into() };\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "Config", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit for Rust struct")
	}
	if !strings.Contains(result, "Config") {
		t.Error("expected symbol name in result")
	}
	if !strings.Contains(result, "struct") {
		t.Errorf("expected kind 'struct', got:\n%s", result)
	}
}

func TestSearchCode_RustUsesAndCallers(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs":  "pub fn process(data: &str) -> String {\n    data.to_string()\n}\n",
		"main.rs": "use crate::process;\nlet result = process(\"hello\");\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "process", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Uses") {
		t.Errorf("expected Uses section, got:\n%s", result)
	}
	if !strings.Contains(result, "Callers") {
		t.Errorf("expected Callers section, got:\n%s", result)
	}
}

func TestSearchCode_RustFallbackToText(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs": "fn main() {}\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "NonExistent12345", Path: dir, FileType: "rs"})
	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected fallback with no matches, got: %s", result)
	}
}

func TestSearchCode_RustNoRefs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs": "pub fn unused_fn() {}\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "unused_fn", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "No references found") {
		t.Errorf("expected 'No references found', got:\n%s", result)
	}
}

func TestSearchCode_RustTestSeparation_TestsDir(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"lib.rs":                    "pub fn process(data: &str) -> String {\n    data.to_string()\n}\n",
		"main.rs":                   "let result = process(\"hello\");\n",
		"tests/integration_test.rs": "let result = process(\"test\");\n",
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: "process", Path: dir, FileType: "rs"})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit")
	}
	if !strings.Contains(result, "Related Tests") {
		t.Errorf("expected Related Tests for tests/ dir, got:\n%s", result)
	}
}
