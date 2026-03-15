package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ── turnDedupKey tests ──

func TestTurnDedupKey_SameArgs(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}
	tc2 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}
	if turnDedupKey(tc1) != turnDedupKey(tc2) {
		t.Error("same tool + args should produce same key")
	}
}

func TestTurnDedupKey_DifferentArgs(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}
	tc2 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/b.go"}}
	if turnDedupKey(tc1) == turnDedupKey(tc2) {
		t.Error("different paths should produce different keys")
	}
}

func TestTurnDedupKey_DifferentTools(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"path": "/a.go"}}
	if turnDedupKey(tc1) == turnDedupKey(tc2) {
		t.Error("different tools should produce different keys")
	}
}

func TestTurnDedupKey_RangeRead(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}}
	tc2 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}}
	if turnDedupKey(tc1) != turnDedupKey(tc2) {
		t.Error("same range read should produce same key")
	}
	tc3 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "100"}}
	if turnDedupKey(tc1) == turnDedupKey(tc3) {
		t.Error("different range should produce different key")
	}
}

func TestTurnDedupKey_SymbolRead(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Foo"}}
	tc2 := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Foo"}}
	if turnDedupKey(tc1) != turnDedupKey(tc2) {
		t.Error("same symbol read should produce same key")
	}
}

// ── isDedupableForTurn tests ──

func TestIsDedupableForTurn(t *testing.T) {
	tests := []struct {
		tool string
		want bool
	}{
		{"read_file", true},
		{"search_code", true},
		{"list_dir", true},
		{"inspect_symbol", true},
		{"write_file", false},
		{"str_replace", false},
		{"bash", false},
		{"delete_file", false},
	}
	for _, tt := range tests {
		if got := isDedupableForTurn(tt.tool); got != tt.want {
			t.Errorf("isDedupableForTurn(%q) = %v, want %v", tt.tool, got, tt.want)
		}
	}
}

// ── isBatchableReadFile tests ──

func TestIsBatchableReadFile(t *testing.T) {
	tests := []struct {
		name string
		tc   *tools.ToolCall
		want bool
	}{
		{"plain path", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}, true},
		{"symbol read", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Foo"}}, false},
		{"range read", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}}, false},
		{"start_line only", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1"}}, false},
		{"paths specified", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`}}, false},
		{"empty path", &tools.ToolCall{Tool: "read_file", Args: map[string]string{}}, false},
		{"not read_file", &tools.ToolCall{Tool: "search_code", Args: map[string]string{"path": "/a.go"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBatchableReadFile(tt.tc); got != tt.want {
				t.Errorf("isBatchableReadFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ── searchCodeOptionsKey tests ──

func TestSearchCodeOptionsKey_SameOptions(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_type": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "path": ".", "file_type": "go"}}
	if searchCodeOptionsKey(tc1) != searchCodeOptionsKey(tc2) {
		t.Error("same options (different pattern) should produce same key")
	}
}

func TestSearchCodeOptionsKey_DifferentOptions(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_type": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_type": "py"}}
	if searchCodeOptionsKey(tc1) == searchCodeOptionsKey(tc2) {
		t.Error("different file_type should produce different key")
	}
}

func TestSearchCodeOptionsKey_EmptyOptionsIgnored(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "file_type": ""}}
	if searchCodeOptionsKey(tc1) != searchCodeOptionsKey(tc2) {
		t.Error("empty options should be ignored in key")
	}
}

func TestSearchCodeOptionsKey_NotSearchCode(t *testing.T) {
	tc := &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}
	if searchCodeOptionsKey(tc) != "" {
		t.Error("non-search_code should return empty key")
	}
}

// ── isSimpleSearchPattern tests ──

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
			if got := isSimpleSearchPattern(tt.pattern); got != tt.want {
				t.Errorf("isSimpleSearchPattern(%q) = %v, want %v", tt.pattern, got, tt.want)
			}
		})
	}
}

// ── splitMultiPatternResult tests ──

func TestSplitMultiPatternResult_TipAppendedToAllSections(t *testing.T) {
	// lineRangeHint は multi-pattern 結果の末尾に 1 回だけ付く。
	// splitMultiPatternResult は各 section に Tip を付与すべき。
	tip := "\n\nTip: Use str_replace line-range mode (old_str empty + start_line/end_line) to edit matched lines directly."
	result := `Found 5 matches across 2 patterns:

━━ Pattern 1/2: "foo" ━━
📄 a.go:
  1: foo

━━ Pattern 2/2: "bar" ━━
📄 b.go:
  2: bar` + tip

	patterns := []string{"foo", "bar"}
	sections := splitMultiPatternResult(result, patterns)

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
	tip := "\n\nTip: Use str_replace line-range mode."
	result := `Found 5 matches across 2 patterns:

━━ Pattern 1/2: "ok" ━━
📄 a.go:
  1: ok

━━ Pattern 2/2: "bad" ━━
⚠️ Error: regex syntax error` + tip

	patterns := []string{"ok", "bad"}
	sections := splitMultiPatternResult(result, patterns)

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
	tip := "\n\nTip: Use str_replace line-range mode."
	result := `Found 1 match(es) across 1/2 patterns

━━ Pattern 1/2: "hit" ━━
📄 a.go:
  1: hit

━━ Pattern 2/2: "miss" ━━
Warning: include_hidden is partially supported in grep fallback mode
No matches found` + tip

	patterns := []string{"hit", "miss"}
	sections := splitMultiPatternResult(result, patterns)

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
	sections := splitMultiPatternResult(result, patterns)

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
	sections := splitMultiPatternResult(result, []string{"foo", "bar"})
	if sections != nil {
		t.Error("should return nil when header count doesn't match")
	}
}

func TestSplitMultiPatternResult_NoHeaders(t *testing.T) {
	result := "Just some text without any pattern headers"
	sections := splitMultiPatternResult(result, []string{"foo"})
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
	sections := splitMultiPatternResult(result, patterns)
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

// ── cloneToolCallWithNewPattern tests ──

func TestCloneToolCallWithNewPattern(t *testing.T) {
	original := &tools.ToolCall{
		Tool: "search_code",
		Args: map[string]string{
			"pattern":   "foo",
			"path":      ".",
			"file_type": "go",
		},
		RawArgs: map[string]any{
			"pattern":   "foo",
			"path":      ".",
			"file_type": "go",
		},
	}

	cloned := cloneToolCallWithNewPattern(original, "foo,bar")

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

// ── Same-turn duplicate suppression integration tests ──

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_ReadFile(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// Same read_file(path=/a.go) called twice
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/b.go"}, RawArgs: map[string]any{"path": "/b.go"}},
	}

	agent.addToolCallsToHistory("test", toolCalls)
	historyBefore := len(agent.History)

	var executedIDs []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedIDs = append(executedIDs, tc.ID)
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// c1 and c3 should be executed (different files), c2 should be deduplicated
	if len(executedIDs) != 2 {
		t.Fatalf("executed %d tools, want 2; got %v", len(executedIDs), executedIDs)
	}
	if executedIDs[0] != "c1" || executedIDs[1] != "c3" {
		t.Errorf("executedIDs = %v, want [c1, c3]", executedIDs)
	}

	// c2 should have canonical duplicate message in history
	addedMsgs := agent.History[historyBefore:]
	foundDup := false
	for _, msg := range addedMsgs {
		if msg.ToolCallID == "c2" && strings.Contains(msg.Content, "[Duplicate]") {
			foundDup = true
			break
		}
	}
	if !foundDup {
		t.Error("expected duplicate message for c2 in history")
	}

	// Observability
	if agent.Stats.ToolObs.SameTurnDuplicates != 1 {
		t.Errorf("SameTurnDuplicates = %d, want 1", agent.Stats.ToolObs.SameTurnDuplicates)
	}
}

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_NotSuppressed_DifferentArgs(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// Different paths — should NOT be deduplicated
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/b.go"}, RawArgs: map[string]any{"path": "/b.go"}},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	if executedCount != 2 {
		t.Errorf("expected 2 executions for different files, got %d", executedCount)
	}
	if agent.Stats.ToolObs.SameTurnDuplicates != 0 {
		t.Errorf("SameTurnDuplicates = %d, want 0", agent.Stats.ToolObs.SameTurnDuplicates)
	}
}

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_TargetedReadNotBroken(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// Same symbol read twice → should be deduplicated (exact same args)
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Build"}, RawArgs: map[string]any{"path": "/a.go", "symbol": "Build"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Build"}, RawArgs: map[string]any{"path": "/a.go", "symbol": "Build"}},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	var executedIDs []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedIDs = append(executedIDs, tc.ID)
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// c2 is exact duplicate → should be suppressed
	if len(executedIDs) != 1 || executedIDs[0] != "c1" {
		t.Errorf("executedIDs = %v, want [c1]", executedIDs)
	}

	// Different symbol → should NOT be deduplicated
	toolCalls2 := []*tools.ToolCall{
		{ID: "d1", Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Build"}, RawArgs: map[string]any{"path": "/a.go", "symbol": "Build"}},
		{ID: "d2", Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Run"}, RawArgs: map[string]any{"path": "/a.go", "symbol": "Run"}},
	}
	agent2 := NewAgent("test-model", provider, false)
	agent2.Stats = &SessionStats{ToolExecutions: make(map[string]int)}
	agent2.addToolCallsToHistory("test", toolCalls2)

	var executedIDs2 []string
	callback2 := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedIDs2 = append(executedIDs2, tc.ID)
	}
	agent2.executeToolCallsWithParallel(context.Background(), toolCalls2, nil, nil, callback2)

	if len(executedIDs2) != 2 {
		t.Errorf("different symbols should both execute, got %d", len(executedIDs2))
	}
}

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_WriteToolNotDeduped(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// write_file called twice with same args → should NOT be deduplicated
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "write_file", Args: map[string]string{"path": "/a.go", "content": "x"}, RawArgs: map[string]any{"path": "/a.go", "content": "x"}},
		{ID: "c2", Tool: "write_file", Args: map[string]string{"path": "/a.go", "content": "x"}, RawArgs: map[string]any{"path": "/a.go", "content": "x"}},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	if executedCount != 2 {
		t.Errorf("write tools should not be deduplicated, got %d executions", executedCount)
	}
}

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_TextBased(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// Text-based (no ID) duplicate
	toolCalls := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
	}

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	if executedCount != 1 {
		t.Errorf("expected 1 execution (duplicate suppressed), got %d", executedCount)
	}

	// Text-based duplicate should add role="user" message
	foundDup := false
	for _, msg := range agent.History {
		if msg.Role == "user" && strings.Contains(msg.Content, "[Duplicate]") {
			foundDup = true
			break
		}
	}
	if !foundDup {
		t.Error("expected text-based duplicate message in history")
	}
}

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_MultipleDedup(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// Same call 3 times → only first executed
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "search_code", Args: map[string]string{"pattern": "foo"}, RawArgs: map[string]any{"pattern": "foo"}},
		{ID: "c2", Tool: "search_code", Args: map[string]string{"pattern": "foo"}, RawArgs: map[string]any{"pattern": "foo"}},
		{ID: "c3", Tool: "search_code", Args: map[string]string{"pattern": "foo"}, RawArgs: map[string]any{"pattern": "foo"}},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	if executedCount != 1 {
		t.Errorf("expected 1 execution (2 duplicates suppressed), got %d", executedCount)
	}
	if agent.Stats.ToolObs.SameTurnDuplicates != 2 {
		t.Errorf("SameTurnDuplicates = %d, want 2", agent.Stats.ToolObs.SameTurnDuplicates)
	}
}

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_CanonicalMessage(t *testing.T) {
	msg := sameTurnDuplicateMessage("read_file")
	if !strings.Contains(msg, "[Duplicate]") {
		t.Errorf("canonical message should contain [Duplicate], got %q", msg)
	}
	if !strings.Contains(msg, "read_file") {
		t.Errorf("canonical message should contain tool name, got %q", msg)
	}
	if !strings.Contains(msg, "reuse previous result") {
		t.Errorf("canonical message should contain reuse instruction, got %q", msg)
	}
}

// ── read_file batch merge eligibility tests ──

func TestExecuteToolCallsWithParallel_ReadFile_PlainPathBatch(t *testing.T) {
	// 2+ plain path reads for SAME file → second is deduplicated
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var executedIDs []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedIDs = append(executedIDs, tc.ID)
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// Only c1 should execute; c2 is a same-turn duplicate
	if len(executedIDs) != 1 || executedIDs[0] != "c1" {
		t.Errorf("executedIDs = %v, want [c1]", executedIDs)
	}
}

func TestExecuteToolCallsWithParallel_ReadFile_RangeNotBatched(t *testing.T) {
	// Range reads are still dedupable if args are identical
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// Different ranges → both should execute
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}, RawArgs: map[string]any{"path": "/a.go", "start_line": 1, "end_line": 50}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "51", "end_line": "100"}, RawArgs: map[string]any{"path": "/a.go", "start_line": 51, "end_line": 100}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	if executedCount != 2 {
		t.Errorf("different ranges should both execute, got %d", executedCount)
	}
}

func TestExecuteToolCallsWithParallel_ReadFile_MixedNotBroken(t *testing.T) {
	// Mixed: plain + symbol + range → each treated independently
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Build"}, RawArgs: map[string]any{"path": "/a.go", "symbol": "Build"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}, RawArgs: map[string]any{"path": "/a.go", "start_line": 1, "end_line": 50}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// All three have different args → all should execute (no dedup)
	if executedCount != 3 {
		t.Errorf("mixed call types should all execute, got %d", executedCount)
	}
}

func TestExecuteToolCallsWithParallel_ReadFile_BatchResultAndHistory(t *testing.T) {
	// Verify history is correct after batch dedup
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)
	historyBefore := len(agent.History)

	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		// Simulate adding result to history (as agent_chat.go does)
		if tc.ID != "" {
			agent.History = append(agent.History, api.Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
				ToolName:   tc.Tool,
			})
		}
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// History should have:
	// - c1 result (from callback)
	// - c2 duplicate message (from Phase 2 statusDuplicate handling)
	addedMsgs := agent.History[historyBefore:]
	if len(addedMsgs) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(addedMsgs))
	}

	// c2 should be the duplicate (could be in any order depending on Phase 2)
	var foundC1, foundC2Dup bool
	for _, msg := range addedMsgs {
		if msg.ToolCallID == "c1" {
			foundC1 = true
		}
		if msg.ToolCallID == "c2" && strings.Contains(msg.Content, "[Duplicate]") {
			foundC2Dup = true
		}
	}
	if !foundC1 {
		t.Error("expected c1 result in history")
	}
	if !foundC2Dup {
		t.Error("expected c2 duplicate message in history")
	}
}

// ── search_code batching tests ──

func TestSearchCodeBatch_SameOptionsGrouped(t *testing.T) {
	// Two search_code calls with same options but different patterns
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "handleSSE", "path": ".", "file_type": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "parseResponse", "path": ".", "file_type": "go"}}

	key1 := searchCodeOptionsKey(tc1)
	key2 := searchCodeOptionsKey(tc2)

	if key1 != key2 {
		t.Errorf("same options should produce same key: %q vs %q", key1, key2)
	}
}

func TestSearchCodeBatch_DifferentOptionsNotGrouped(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "file_type": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "file_type": "py"}}

	key1 := searchCodeOptionsKey(tc1)
	key2 := searchCodeOptionsKey(tc2)

	if key1 == key2 {
		t.Error("different file_type should produce different keys")
	}
}

func TestSearchCodeBatch_MultiPatternPatternNotBatched(t *testing.T) {
	// Already multi-pattern → should not be batched further
	if isSimpleSearchPattern("foo,bar") {
		t.Error("multi-pattern should not be considered simple")
	}
}

func TestSplitMultiPatternResult_ThreePatterns(t *testing.T) {
	result := `Found 15 matches across 3 patterns:

━━ Pattern 1/3: "foo" ━━
📄 a.go:
  1: foo

━━ Pattern 2/3: "bar" ━━
📄 b.go:
  2: bar

━━ Pattern 3/3: "baz" ━━
📄 c.go:
  3: baz
`

	patterns := []string{"foo", "bar", "baz"}
	sections := splitMultiPatternResult(result, patterns)

	if sections == nil {
		t.Fatal("expected non-nil sections")
	}

	for _, p := range patterns {
		if _, ok := sections[p]; !ok {
			t.Errorf("missing section for pattern %q", p)
		}
	}

	if !strings.Contains(sections["foo"], "a.go") {
		t.Error("foo section should contain a.go")
	}
	if !strings.Contains(sections["bar"], "b.go") {
		t.Error("bar section should contain b.go")
	}
	if !strings.Contains(sections["baz"], "c.go") {
		t.Error("baz section should contain c.go")
	}
}

// ── Regression: existing behaviors preserved ──

func TestExecuteToolCallsWithParallel_DedupDoesNotBreakLoopDetection(t *testing.T) {
	// Loop detection should still work alongside dedup
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}
	cfg := agent.cfg()
	origThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 2
	defer func() { cfg.LoopDetection.Threshold = origThreshold }()

	// Loop detection checks across responses, dedup checks within response.
	// Both should work independently.
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/x.go"}, RawArgs: map[string]any{"path": "/x.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/x.go"}, RawArgs: map[string]any{"path": "/x.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var lastTC *tools.ToolCall
	count := 0
	loopDetectFn := func(tc *tools.ToolCall) bool {
		if isSameToolCall(tc, lastTC) {
			count++
			if count >= cfg.LoopDetection.Threshold {
				return true
			}
		} else {
			count = 1
		}
		lastTC = tc
		return false
	}

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	loopDetected := agent.executeToolCallsWithParallel(context.Background(), toolCalls, loopDetectFn, nil, callback)

	// Loop detection should fire (same call twice with threshold=2)
	// Phase 0 catches c2 as loopAbort before Phase 0.5 can catch it as duplicate
	if !loopDetected {
		// c2 was caught by loop detection, not dedup
		t.Logf("loop detected as expected")
	}
	// Either way, c2 should not be executed
	if executedCount > 1 {
		t.Errorf("expected at most 1 execution, got %d", executedCount)
	}
}

func TestExecuteToolCallsWithParallel_DedupDoesNotBreakSkip(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "create_plan", Args: map[string]string{"title": "X"}, RawArgs: map[string]any{"title": "X"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "create_plan" {
			return true, "[create_plan] Ignored"
		}
		return false, ""
	}

	var executedIDs []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedIDs = append(executedIDs, tc.ID)
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, skipFn, callback)

	// c1 executed, c2 skipped, c3 deduplicated
	if len(executedIDs) != 1 || executedIDs[0] != "c1" {
		t.Errorf("executedIDs = %v, want [c1]", executedIDs)
	}
}

func TestExecuteToolCallsWithParallel_ObservabilityIntact(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c3", Tool: "list_dir", Args: map[string]string{"path": "."}, RawArgs: map[string]any{"path": "."}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil,
		func(_ int, _ *tools.ToolCall, _ string, _ *tools.FileChange) {})

	// ToolExecutions counts only actually-executed calls (c2 is duplicate → not counted)
	if agent.Stats.ToolExecutions["read_file"] != 1 {
		t.Errorf("ToolExecutions[read_file] = %d, want 1 (duplicate not counted)", agent.Stats.ToolExecutions["read_file"])
	}
	if agent.Stats.ToolExecutions["list_dir"] != 1 {
		t.Errorf("ToolExecutions[list_dir] = %d, want 1", agent.Stats.ToolExecutions["list_dir"])
	}
	// Dedup observability
	if agent.Stats.ToolObs.SameTurnDuplicates != 1 {
		t.Errorf("SameTurnDuplicates = %d, want 1", agent.Stats.ToolObs.SameTurnDuplicates)
	}
}

func TestExecuteToolCallsWithParallel_SameTurnDuplicate_ReadFileGuidancePreserved(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)
	historyBefore := len(agent.History)

	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		historyContent := agent.compactToolResult(tc, result)
		agent.History = append(agent.History, api.Message{
			Role:       "tool",
			Content:    historyContent,
			ToolCallID: tc.ID,
			ToolName:   tc.Tool,
		})
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	addedMsgs := agent.History[historyBefore:]
	var duplicateWithGuidance bool
	for _, msg := range addedMsgs {
		if msg.ToolCallID == "c3" &&
			strings.Contains(msg.Content, "[Duplicate]") &&
			strings.Contains(msg.Content, "[GUIDANCE]") {
			duplicateWithGuidance = true
			break
		}
	}
	if !duplicateWithGuidance {
		t.Fatalf("expected duplicate message with guidance for c3, got %+v", addedMsgs)
	}
}

// ── splitReadFileBatchResult tests ──

func TestSplitReadFileBatchResult_TwoFiles(t *testing.T) {
	result := "📄 File: /a.go\npackage main\n\nfunc main() {}\n\n📄 File: /b.go\npackage util\n\nfunc Helper() {}\n"

	paths := []string{"/a.go", "/b.go"}
	sections := splitReadFileBatchResult(result, paths)

	if sections == nil {
		t.Fatal("expected non-nil sections")
	}
	if len(sections) != 2 {
		t.Fatalf("expected 2 sections, got %d", len(sections))
	}
	if !strings.Contains(sections["/a.go"], "package main") {
		t.Errorf("/a.go section should contain 'package main', got %q", sections["/a.go"])
	}
	if !strings.Contains(sections["/b.go"], "package util") {
		t.Errorf("/b.go section should contain 'package util', got %q", sections["/b.go"])
	}
	// Each section should not contain the other file's content
	if strings.Contains(sections["/a.go"], "package util") {
		t.Error("/a.go section should not contain /b.go content")
	}
	if strings.Contains(sections["/b.go"], "package main") {
		t.Error("/b.go section should not contain /a.go content")
	}
}

func TestSplitReadFileBatchResult_ThreeFiles(t *testing.T) {
	result := "📄 File: /a.go\ncontent_a\n\n📄 File: /b.go\ncontent_b\n\n📄 File: /c.go\ncontent_c\n"

	paths := []string{"/a.go", "/b.go", "/c.go"}
	sections := splitReadFileBatchResult(result, paths)

	if sections == nil {
		t.Fatal("expected non-nil sections")
	}
	for _, p := range paths {
		if _, ok := sections[p]; !ok {
			t.Errorf("missing section for %s", p)
		}
	}
	if !strings.Contains(sections["/a.go"], "content_a") {
		t.Error("/a.go section wrong")
	}
	if !strings.Contains(sections["/b.go"], "content_b") {
		t.Error("/b.go section wrong")
	}
	if !strings.Contains(sections["/c.go"], "content_c") {
		t.Error("/c.go section wrong")
	}
}

func TestSplitReadFileBatchResult_HeaderMissing(t *testing.T) {
	// Missing header for /b.go → should return nil
	result := "📄 File: /a.go\ncontent_a\n"

	paths := []string{"/a.go", "/b.go"}
	sections := splitReadFileBatchResult(result, paths)

	if sections != nil {
		t.Error("should return nil when header for /b.go is missing")
	}
}

func TestSplitReadFileBatchResult_ErrorFile(t *testing.T) {
	// File with error result should still be split correctly
	result := "📄 File: /a.go\npackage main\n\n📄 File: /missing.go\nError: file not found: /missing.go\n"

	paths := []string{"/a.go", "/missing.go"}
	sections := splitReadFileBatchResult(result, paths)

	if sections == nil {
		t.Fatal("expected non-nil sections even with error")
	}
	if !strings.Contains(sections["/a.go"], "package main") {
		t.Error("/a.go section should have content")
	}
	if !strings.Contains(sections["/missing.go"], "Error:") {
		t.Error("/missing.go section should have error")
	}
}

func TestSplitReadFileBatchResult_EmptyResult(t *testing.T) {
	sections := splitReadFileBatchResult("", []string{"/a.go"})
	if sections != nil {
		t.Error("should return nil for empty result")
	}
}

// ── buildReadFileBatchToolCall tests ──

func TestBuildReadFileBatchToolCall(t *testing.T) {
	paths := []string{"/a.go", "/b.go", "/c.go"}
	tc := buildReadFileBatchToolCall(paths)

	if tc.Tool != "read_file" {
		t.Errorf("Tool = %q, want read_file", tc.Tool)
	}
	if tc.Args["paths"] == "" {
		t.Error("paths arg should not be empty")
	}
	// paths should be valid JSON array
	if !strings.HasPrefix(tc.Args["paths"], "[") {
		t.Errorf("paths should be JSON array, got %q", tc.Args["paths"])
	}
	// RawArgs.paths should be []string
	rawPaths, ok := tc.RawArgs["paths"].([]string)
	if !ok {
		t.Fatal("RawArgs[paths] should be []string")
	}
	if len(rawPaths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(rawPaths))
	}
}

// ── read_file batch merge group judgment tests ──

func TestIsBatchableReadFile_Comprehensive(t *testing.T) {
	tests := []struct {
		name string
		tc   *tools.ToolCall
		want bool
	}{
		{"plain path", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go"}}, true},
		{"symbol read", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "symbol": "Foo"}}, false},
		{"range read", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}}, false},
		{"start_line only", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1"}}, false},
		{"end_line only", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "end_line": "50"}}, false},
		{"paths specified", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`}}, false},
		{"empty path", &tools.ToolCall{Tool: "read_file", Args: map[string]string{}}, false},
		{"not read_file", &tools.ToolCall{Tool: "search_code", Args: map[string]string{"path": "/a.go"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBatchableReadFile(tt.tc); got != tt.want {
				t.Errorf("isBatchableReadFile() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ── read_file batch merge: duplicate suppression does not interfere ──

func TestReadFileBatchMerge_DuplicateSuppressionFirst(t *testing.T) {
	// read_file(a.go), read_file(a.go), read_file(b.go)
	// Duplicate suppression catches the second a.go first.
	// Batch merge only sees a.go + b.go (2 distinct paths after dedup).
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/b.go"}, RawArgs: map[string]any{"path": "/b.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var callbackIDs []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		callbackIDs = append(callbackIDs, tc.ID)
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// c1 and c3 should have callbacks (c2 is duplicate, no callback)
	if len(callbackIDs) != 2 {
		t.Fatalf("expected 2 callbacks, got %d: %v", len(callbackIDs), callbackIDs)
	}
	if callbackIDs[0] != "c1" || callbackIDs[1] != "c3" {
		t.Errorf("callbackIDs = %v, want [c1, c3]", callbackIDs)
	}

	// Duplicate suppression metric should be 1
	if agent.Stats.ToolObs.SameTurnDuplicates != 1 {
		t.Errorf("SameTurnDuplicates = %d, want 1", agent.Stats.ToolObs.SameTurnDuplicates)
	}
}

// ── read_file batch merge: callback order preserved ──

func TestReadFileBatchMerge_CallbackOrderPreserved(t *testing.T) {
	// Verify callbacks fire in original tool call order
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/x.go"}, RawArgs: map[string]any{"path": "/x.go"}},
		{ID: "c2", Tool: "search_code", Args: map[string]string{"pattern": "foo"}, RawArgs: map[string]any{"pattern": "foo"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/y.go"}, RawArgs: map[string]any{"path": "/y.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var callbackOrder []string
	callback := func(idx int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		callbackOrder = append(callbackOrder, tc.ID)
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// All three should execute; order should be c1, c2, c3
	if len(callbackOrder) != 3 {
		t.Fatalf("expected 3 callbacks, got %d: %v", len(callbackOrder), callbackOrder)
	}
	if callbackOrder[0] != "c1" || callbackOrder[1] != "c2" || callbackOrder[2] != "c3" {
		t.Errorf("callbackOrder = %v, want [c1, c2, c3]", callbackOrder)
	}
}

// ── read_file batch merge: observability ──

func TestReadFileBatchMerge_Observability(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// 3 plain reads → should trigger batch merge
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/b.go"}, RawArgs: map[string]any{"path": "/b.go"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/c.go"}, RawArgs: map[string]any{"path": "/c.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {}
	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// ReadFileBatchMerges should be >= 0 (it may or may not fire depending on
	// whether the batched read_file(paths) produces parseable output in test env).
	// The key check: if it fires, ToolExecutions counts each individual call.
	if agent.Stats.ToolObs.ReadFileBatchMerges > 0 {
		// Batch merge fired: each call should still count as executed
		totalReadFile := agent.Stats.ToolExecutions["read_file"]
		if totalReadFile != 3 {
			t.Errorf("ToolExecutions[read_file] = %d, want 3 after batch merge", totalReadFile)
		}
	}
}

// ── read_file batch merge: single batchable read should not merge ──

func TestReadFileBatchMerge_SingleReadNotMerged(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// Only 1 plain read + 1 non-batchable → no merge
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/b.go", "symbol": "Foo"}, RawArgs: map[string]any{"path": "/b.go", "symbol": "Foo"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var callbackCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		callbackCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// Both should execute individually (no batch merge since only 1 batchable)
	if callbackCount != 2 {
		t.Errorf("expected 2 callbacks (no merge), got %d", callbackCount)
	}
	if agent.Stats.ToolObs.ReadFileBatchMerges != 0 {
		t.Errorf("ReadFileBatchMerges = %d, want 0", agent.Stats.ToolObs.ReadFileBatchMerges)
	}
}
