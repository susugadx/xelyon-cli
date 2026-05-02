package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ── toolruntime.IsBatchableReadFile tests ──

func TestIsBatchableReadFile(t *testing.T) {
	tests := []struct {
		name string
		tc   *tools.ToolCall
		want bool
	}{
		{"plain path", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}, true},
		{"range read", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}}, false},
		{"start_line only", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1"}}, false},
		{"paths single", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go"]`}}, true},
		{"paths specified", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`}}, true},
		{"paths detail auto", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`, "detail": "auto"}}, true},
		{"paths detail outline", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`, "detail": "outline"}}, false},
		{"paths detail compact", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`, "detail": "compact"}}, false},
		{"paths with range", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go:1-20","b.go"]`}}, false},
		{"empty path", &tools.ToolCall{Tool: "read_file", Args: map[string]string{}}, false},
		{"not read_file", &tools.ToolCall{Tool: "search_code", Args: map[string]string{"path": "/a.go"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := toolruntime.IsBatchableReadFile(tt.tc); got != tt.want {
				t.Errorf("toolruntime.IsBatchableReadFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ── toolruntime.SearchCodeOptionsKey tests ──

func TestSearchCodeOptionsKey_SameOptions(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_filter": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "path": ".", "file_filter": "go"}}
	if toolruntime.SearchCodeOptionsKey(tc1) != toolruntime.SearchCodeOptionsKey(tc2) {
		t.Error("same options (different pattern) should produce same key")
	}
}

func TestSearchCodeOptionsKey_DifferentOptions(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_filter": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_filter": "py"}}
	if toolruntime.SearchCodeOptionsKey(tc1) == toolruntime.SearchCodeOptionsKey(tc2) {
		t.Error("different file_filter should produce different key")
	}
}

func TestSearchCodeOptionsKey_EmptyOptionsIgnored(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "file_filter": ""}}
	if toolruntime.SearchCodeOptionsKey(tc1) != toolruntime.SearchCodeOptionsKey(tc2) {
		t.Error("empty options should be ignored in key")
	}
}

func TestSearchCodeOptionsKey_NormalizesModeAndLegacyRegex(t *testing.T) {
	autoImplicit := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo"}}
	autoExplicit := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "mode": "auto"}}
	if toolruntime.SearchCodeOptionsKey(autoImplicit) != toolruntime.SearchCodeOptionsKey(autoExplicit) {
		t.Error("implicit auto and explicit auto should produce same key")
	}

	regexLegacy := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "is_regex": "true"}}
	regexMode := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "mode": "regex"}}
	if toolruntime.SearchCodeOptionsKey(regexLegacy) != toolruntime.SearchCodeOptionsKey(regexMode) {
		t.Error("legacy is_regex=true and mode=regex should produce same key")
	}
}

func TestSearchCodeOptionsKey_NotSearchCode(t *testing.T) {
	tc := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}
	if toolruntime.SearchCodeOptionsKey(tc) != "" {
		t.Error("non-search_code should return empty key")
	}
}

// ── toolruntime.IsSimpleSearchPattern tests ──

func TestIsSimpleSearchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{"foo", true},
		{"handleSSE", true},
		{"func.*Build", true},
		{"foo,bar", false},     // multi-pattern
		{`foo\,bar`, true},     // escaped comma = literal
		{`a\,b,c`, false},      // has unescaped comma
		{"", true},             // empty is simple
		{"struct{}", true},     // no comma
		{"a,b,c,d,e,f", false}, // many patterns
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			if got := toolruntime.IsSimpleSearchPattern(tt.pattern); got != tt.want {
				t.Errorf("toolruntime.IsSimpleSearchPattern(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

// ── toolruntime.SplitMultiPatternResult tests ──

func TestSplitMultiPatternResult_TipAppendedToAllSections(t *testing.T) {
	// lineRangeHint は multi-pattern 結果の末尾に 1 回だけ付く。
	// toolruntime.SplitMultiPatternResult は各 section に Tip を付与すべき。
	tip := "\n\nTip: Use the active edit tool with the matched lines plus surrounding context to make exact edits."
	result := `Found 5 matches across 2 patterns:

━━ Pattern 1/2: "foo" ━━
📄 a.go:
  1: foo

━━ Pattern 2/2: "bar" ━━
📄 b.go:
  2: bar` + tip

	patterns := []string{"foo", "bar"}
	sections := toolruntime.SplitMultiPatternResult(result, patterns)

	if sections == nil {
		t.Fatal("expected non-nil sections")
	}

	// Both sections should have the Tip appended
	for _, p := range patterns {
		if !strings.Contains(sections[p], "Tip:") {
			t.Errorf("section for %q missing Tip hint:\n%s", p, sections[p])
		}
	}
}

func TestSplitMultiPatternResult_ErrorSectionNoTip(t *testing.T) {
	// Error section should NOT get the Tip
	tip := "\n\nTip: Use the active edit tool."
	result := `Found 5 matches across 2 patterns:

━━ Pattern 1/2: "ok" ━━
📄 a.go:
  1: ok

━━ Pattern 2/2: "bad" ━━
⚠️ Error: regex syntax error` + tip

	patterns := []string{"ok", "bad"}
	sections := toolruntime.SplitMultiPatternResult(result, patterns)

	if sections == nil {
		t.Fatal("expected non-nil sections")
	}
	if !strings.Contains(sections["ok"], "Tip:") {
		t.Error("ok section should have Tip")
	}
	if strings.Contains(sections["bad"], "Tip:") {
		t.Error("error section should NOT have Tip")
	}
}

func TestSplitMultiPatternResult_NoMatchesSectionNoTip(t *testing.T) {
	tip := "\n\nTip: Use the active edit tool."
	result := `Found 1 match(es) across 1/2 patterns

━━ Pattern 1/2: "hit" ━━
📄 a.go:
  1: hit

━━ Pattern 2/2: "miss" ━━
Warning: include_hidden is partially supported in grep fallback mode
No matches found` + tip

	patterns := []string{"hit", "miss"}
	sections := toolruntime.SplitMultiPatternResult(result, patterns)

	if sections == nil {
		t.Fatal("expected non-nil sections")
	}
	if !strings.Contains(sections["hit"], "Tip:") {
		t.Error("match section should have Tip")
	}
	if strings.Contains(sections["miss"], "Tip:") {
		t.Errorf("no-match section should NOT have Tip:\n%s", sections["miss"])
	}
	if !strings.Contains(sections["miss"], "No matches found") {
		t.Errorf("no-match section should preserve no-match message:\n%s", sections["miss"])
	}
}

func TestSplitMultiPatternResult_TwoPatterns(t *testing.T) {
	result := `Found 10 matches across 2 patterns:

━━ Pattern 1/2: "handleSSE" ━━
📄 stream.go:
  42: func handleSSE() {

━━ Pattern 2/2: "parseResponse" ━━
📄 api.go:
  15: func parseResponse() {
`

	patterns := []string{"handleSSE", "parseResponse"}
	sections := toolruntime.SplitMultiPatternResult(result, patterns)

	if sections == nil {
		t.Fatal("expected non-nil sections")
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}

	if !strings.Contains(sections["handleSSE"], "stream.go") {
		t.Errorf("handleSSE section should contain stream.go, got %q", sections["handleSSE"])
	}
	if !strings.Contains(sections["parseResponse"], "api.go") {
		t.Errorf("parseResponse section should contain api.go, got %q", sections["parseResponse"])
	}
}

func TestSplitMultiPatternResult_HeaderCountMismatch(t *testing.T) {
	// Only 1 header but 2 expected patterns → should return nil
	result := `━━ Pattern 1/1: "foo" ━━
some result
`
	sections := toolruntime.SplitMultiPatternResult(result, []string{"foo", "bar"})
	if sections != nil {
		t.Error("should return nil when header count doesn't match")
	}
}

func TestSplitMultiPatternResult_NoHeaders(t *testing.T) {
	result := "Just some text without any pattern headers"
	sections := toolruntime.SplitMultiPatternResult(result, []string{"foo"})
	if sections != nil {
		t.Error("should return nil when no headers found")
	}
}

func TestSplitMultiPatternResult_PatternWithError(t *testing.T) {
	result := `Found 5 matches across 2 patterns:

━━ Pattern 1/2: "validPattern" ━━
📄 file.go:
  10: validPattern

━━ Pattern 2/2: "bad[pattern" ━━
⚠️ Error: regex syntax error
`
	patterns := []string{"validPattern", "bad[pattern"}
	sections := toolruntime.SplitMultiPatternResult(result, patterns)
	if sections == nil {
		t.Fatal("should split even with error sections")
	}
	if !strings.Contains(sections["validPattern"], "file.go") {
		t.Error("valid pattern section should have results")
	}
	if !strings.Contains(sections["bad[pattern"], "Error") {
		t.Error("error pattern section should have error message")
	}
}

// ── toolruntime.CloneToolCallWithNewPattern tests ──

func TestCloneToolCallWithNewPattern(t *testing.T) {
	original := &tools.ToolCall{
		Tool: "search_code",
		Args: map[string]string{
			"pattern":     "foo",
			"path":        ".",
			"file_filter": "go",
		},
		RawArgs: map[string]any{
			"pattern":     "foo",
			"path":        ".",
			"file_filter": "go",
		},
	}

	cloned := toolruntime.CloneToolCallWithNewPattern(original, "foo,bar")

	if cloned.Args["pattern"] != "foo,bar" {
		t.Errorf("cloned pattern = %q, want %q", cloned.Args["pattern"], "foo,bar")
	}
	if cloned.Args["path"] != "." {
		t.Error("other args should be preserved")
	}
	// Verify original is not modified
	if original.Args["pattern"] != "foo" {
		t.Error("original should not be modified")
	}
}
