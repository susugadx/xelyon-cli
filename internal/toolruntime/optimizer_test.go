package toolruntime

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestSegmentReadFileBatchesSplitsAtUnsafeToolsAndSkipsIneligibleReads(t *testing.T) {
	calls := []*tools.ToolCall{
		readFileCall("a.go"),
		readFileCall("b.go"),
		{Tool: "write_file", Args: map[string]string{"path": "a.go"}},
		readFileCall("c.go"),
		readFileCallWithArgs(map[string]string{"path": "d.go", "start_line": "10"}),
		readFileCall("e.go"),
		readFileCall("f.go"),
	}
	execFlags := []bool{true, true, true, true, true, false, true}

	segments := SegmentReadFileBatches(calls, execFlags)
	if len(segments) != 2 {
		t.Fatalf("len(segments) = %d, want 2: %#v", len(segments), segments)
	}
	first := segments[0]
	if !equalInts(first.Indices, []int{0, 1}) {
		t.Fatalf("first segment indices = %#v, want [0 1]", first.Indices)
	}
	if strings.Join(first.Paths, ",") != "a.go,b.go" {
		t.Fatalf("first segment paths = %#v, want a.go,b.go", first.Paths)
	}
	if !equalInts(first.PathCounts, []int{1, 1}) {
		t.Fatalf("first path counts = %#v, want [1 1]", first.PathCounts)
	}
	second := segments[1]
	if !equalInts(second.Indices, []int{3, 6}) {
		t.Fatalf("second segment indices = %#v, want [3 6]", second.Indices)
	}
	if strings.Join(second.Paths, ",") != "c.go,f.go" {
		t.Fatalf("second segment paths = %#v, want c.go,f.go", second.Paths)
	}
}

func TestBuildReadFileBatchToolCallEncodesPathsAndFullBudget(t *testing.T) {
	call := BuildReadFileBatchToolCall([]string{"a.go", "b.go"}, true)

	if call.Tool != "read_file" {
		t.Fatalf("Tool = %q, want read_file", call.Tool)
	}
	if call.Args["paths"] != `["a.go","b.go"]` || call.Args["_full_budget"] != "true" {
		t.Fatalf("Args = %#v, want paths JSON and full budget flag", call.Args)
	}
	paths, ok := call.RawArgs["paths"].([]string)
	if !ok || strings.Join(paths, ",") != "a.go,b.go" || call.RawArgs["_full_budget"] != true {
		t.Fatalf("RawArgs = %#v, want typed paths and full budget flag", call.RawArgs)
	}

	call = BuildReadFileBatchToolCall([]string{"a.go"}, false)
	if _, ok := call.Args["_full_budget"]; ok {
		t.Fatalf("Args = %#v, want no full budget flag", call.Args)
	}
	if _, ok := call.RawArgs["_full_budget"]; ok {
		t.Fatalf("RawArgs = %#v, want no full budget flag", call.RawArgs)
	}
}

func TestSearchCodeOptionsKeyNormalizesModeAndIgnoresPattern(t *testing.T) {
	base := SearchCodeOptionsKey(&tools.ToolCall{Tool: "search_code", Args: map[string]string{
		"pattern": "alpha",
		"path":    "internal",
		"mode":    "REGEX",
	}})
	same := SearchCodeOptionsKey(&tools.ToolCall{Tool: "search_code", Args: map[string]string{
		"pattern":  "beta",
		"path":     "internal",
		"is_regex": "true",
	}})
	if base != same {
		t.Fatalf("options key differs for same non-pattern options: %q != %q", base, same)
	}

	differentPath := SearchCodeOptionsKey(&tools.ToolCall{Tool: "search_code", Args: map[string]string{
		"pattern": "alpha",
		"path":    "cmd",
		"mode":    "regex",
	}})
	if differentPath == base {
		t.Fatalf("path change did not affect options key: %q", base)
	}
	if key := SearchCodeOptionsKey(&tools.ToolCall{Tool: "read_file", Args: map[string]string{"pattern": "x"}}); key != "" {
		t.Fatalf("non-search_code key = %q, want empty", key)
	}
}

func TestIsSimpleSearchPatternTreatsEscapedCommaAsLiteral(t *testing.T) {
	tests := []struct {
		pattern string
		want    bool
	}{
		{pattern: "single", want: true},
		{pattern: `literal\,comma`, want: true},
		{pattern: "alpha,beta", want: false},
		{pattern: `alpha\,beta,gamma`, want: false},
	}

	for _, tt := range tests {
		if got := IsSimpleSearchPattern(tt.pattern); got != tt.want {
			t.Fatalf("IsSimpleSearchPattern(%q) = %v, want %v", tt.pattern, got, tt.want)
		}
	}
}

func TestSplitMultiPatternResultPreservesTipOnlyForSuccessfulSections(t *testing.T) {
	result := strings.Join([]string{
		`━━ Pattern 1/3: "alpha" ━━`,
		"internal/a.go:1:alpha",
		`━━ Pattern 2/3: "beta" ━━`,
		"No matches found",
		`━━ Pattern 3/3: "gamma" ━━`,
		"⚠️ Error: ripgrep failed",
		"",
		"Tip: use path:line-line for focused reads",
	}, "\n")

	sections := SplitMultiPatternResult(result, []string{"alpha", "beta", "gamma"})
	if len(sections) != 3 {
		t.Fatalf("sections = %#v, want 3 entries", sections)
	}
	if !strings.Contains(sections["alpha"], "Tip: use path:line-line") {
		t.Fatalf("alpha section = %q, want trailing Tip", sections["alpha"])
	}
	if strings.Contains(sections["beta"], "Tip:") {
		t.Fatalf("beta no-match section = %q, want no Tip", sections["beta"])
	}
	if strings.Contains(sections["gamma"], "Tip:") {
		t.Fatalf("gamma error section = %q, want no Tip", sections["gamma"])
	}
}

func TestSplitMultiPatternResultReturnsNilOnHeaderMismatch(t *testing.T) {
	result := `━━ Pattern 1/1: "alpha" ━━
internal/a.go:1:alpha`
	if sections := SplitMultiPatternResult(result, []string{"alpha", "beta"}); sections != nil {
		t.Fatalf("sections = %#v, want nil for header mismatch", sections)
	}
}

func TestCloneToolCallWithNewPatternCopiesArgsAndRawArgs(t *testing.T) {
	original := &tools.ToolCall{
		Tool:    "search_code",
		Args:    map[string]string{"pattern": "old", "path": "internal"},
		RawArgs: map[string]any{"pattern": "old", "path": "internal"},
	}

	cloned := CloneToolCallWithNewPattern(original, "new")
	if cloned == original {
		t.Fatal("CloneToolCallWithNewPattern() returned original pointer")
	}
	if cloned.Args["pattern"] != "new" || cloned.RawArgs["pattern"] != "new" {
		t.Fatalf("cloned pattern = args:%v raw:%v, want new", cloned.Args["pattern"], cloned.RawArgs["pattern"])
	}
	cloned.Args["path"] = "mutated"
	if original.Args["path"] != "internal" {
		t.Fatalf("original args mutated: %#v", original.Args)
	}
}

func readFileCall(path string) *tools.ToolCall {
	return readFileCallWithArgs(map[string]string{"path": path})
}

func readFileCallWithArgs(args map[string]string) *tools.ToolCall {
	return &tools.ToolCall{Tool: "read_file", Args: args}
}

func equalInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
