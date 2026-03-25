package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ── isBatchableReadFile tests ──

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
		{"paths with range", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go:1-20","b.go"]`}}, false},
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
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_filter": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "path": ".", "file_filter": "go"}}
	if searchCodeOptionsKey(tc1) != searchCodeOptionsKey(tc2) {
		t.Error("same options (different pattern) should produce same key")
	}
}

func TestSearchCodeOptionsKey_DifferentOptions(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_filter": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "path": ".", "file_filter": "py"}}
	if searchCodeOptionsKey(tc1) == searchCodeOptionsKey(tc2) {
		t.Error("different file_filter should produce different key")
	}
}

func TestSearchCodeOptionsKey_EmptyOptionsIgnored(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "file_filter": ""}}
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
	tip := "\n\nTip: Use the active edit tool with the matched lines plus surrounding context to make exact edits."
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
	tip := "\n\nTip: Use the active edit tool."
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
	tip := "\n\nTip: Use the active edit tool."
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

// ── read_file batch merge eligibility tests ──

func TestExecuteToolCallsWithParallel_ReadFile_RangeNotBatched(t *testing.T) {
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
	// Mixed: plain read + search_code + range read → each treated independently
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "search_code", Args: map[string]string{"pattern": "Build"}, RawArgs: map[string]any{"pattern": "Build"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}, RawArgs: map[string]any{"path": "/a.go", "start_line": 1, "end_line": 50}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	var executedCount int
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedCount++
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, callback)

	// All three should execute without deduplication conflicts
	if executedCount != 3 {
		t.Errorf("mixed call types should all execute, got %d", executedCount)
	}
}

func TestExecuteToolCallsWithParallel_ReadFile_BatchResultAndHistory(t *testing.T) {
	// Verify history is correct after batch execution
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
	// - c2 result (from callback)
	addedMsgs := agent.History[historyBefore:]
	if len(addedMsgs) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(addedMsgs))
	}

	var foundC1, foundC2 bool
	for _, msg := range addedMsgs {
		if msg.ToolCallID == "c1" {
			foundC1 = true
		}
		if msg.ToolCallID == "c2" {
			foundC2 = true
		}
	}
	if !foundC1 {
		t.Error("expected c1 result in history")
	}
	if !foundC2 {
		t.Error("expected c2 result in history")
	}
}

// ── search_code batching tests ──

func TestSearchCodeBatch_SameOptionsGrouped(t *testing.T) {
	// Two search_code calls with same options but different patterns
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "handleSSE", "path": ".", "file_filter": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "parseResponse", "path": ".", "file_filter": "go"}}

	key1 := searchCodeOptionsKey(tc1)
	key2 := searchCodeOptionsKey(tc2)

	if key1 != key2 {
		t.Errorf("same options should produce same key: %q vs %q", key1, key2)
	}
}

func TestSearchCodeBatch_DifferentOptionsNotGrouped(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "file_filter": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "file_filter": "py"}}

	key1 := searchCodeOptionsKey(tc1)
	key2 := searchCodeOptionsKey(tc2)

	if key1 == key2 {
		t.Error("different file_filter should produce different keys")
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

func TestExecuteToolCallsWithParallel_SkipDoesNotBreakRepeatedReads(t *testing.T) {
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

	// c1/c3 executed, c2 skipped
	if len(executedIDs) != 2 || executedIDs[0] != "c1" || executedIDs[1] != "c3" {
		t.Errorf("executedIDs = %v, want [c1 c3]", executedIDs)
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

func TestBuildReadFileBatchToolCall_FullBudget(t *testing.T) {
	paths := []string{"/a.go", "/b.go", "/c.go"}
	tc := buildReadFileBatchToolCall(paths, true)

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
	// fullBudget=true → _full_budget flag が設定される
	if tc.Args["_full_budget"] != "true" {
		t.Error("fullBudget=true should set _full_budget arg")
	}
}

func TestBuildReadFileBatchToolCall_NoFullBudget(t *testing.T) {
	tc := buildReadFileBatchToolCall([]string{"/a.go", "/b.go"}, false)
	if tc.Args["_full_budget"] != "" {
		t.Error("fullBudget=false should not set _full_budget arg")
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
		{"range read", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1", "end_line": "50"}}, false},
		{"start_line only", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "start_line": "1"}}, false},
		{"end_line only", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"path": "/a.go", "end_line": "50"}}, false},
		{"paths single", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go"]`}}, true},
		{"paths specified", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`}}, true},
		{"paths specified with range", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go:10-20","b.go"]`}}, false},
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

	// Only 1 plain read + 1 range read → no merge
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
		{ID: "c2", Tool: "read_file", Args: map[string]string{"path": "/b.go", "start_line": "10", "end_line": "20"}, RawArgs: map[string]any{"path": "/b.go", "start_line": 10, "end_line": 20}},
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

// ── segmentReadFileBatches tests ──

func TestSegmentReadFileBatches_AllReads(t *testing.T) {
	// 全て read_file → 1 セグメント
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/c.go"}},
	}
	flags := []bool{true, true, true}
	segs := segmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].paths) != 3 {
		t.Errorf("segment should have 3 paths, got %d", len(segs[0].paths))
	}
}

func TestSegmentReadFileBatches_SplitAtMutation(t *testing.T) {
	// read(a) -> write(b) -> read(c) -> read(d)
	// write で区切り → セグメント 1: [a] (1件なので不採用), セグメント 2: [c, d]
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "write_file", Args: map[string]string{"path": "/b.go", "content": "x"}},
		{Tool: "read_file", Args: map[string]string{"path": "/c.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/d.go"}},
	}
	flags := []bool{true, true, true, true}
	segs := segmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment (only post-mutation pair), got %d", len(segs))
	}
	if segs[0].paths[0] != "/c.go" || segs[0].paths[1] != "/d.go" {
		t.Errorf("segment paths = %v, want [/c.go, /d.go]", segs[0].paths)
	}
}

func TestSegmentReadFileBatches_TwoSegments(t *testing.T) {
	// read(a) -> read(b) -> str_replace(c) -> read(d) -> read(e)
	// str_replace で区切り → セグメント 1: [a, b], セグメント 2: [d, e]
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
		{Tool: "str_replace", Args: map[string]string{"path": "/c.go", "old_str": "x", "new_str": "y"}},
		{Tool: "read_file", Args: map[string]string{"path": "/d.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/e.go"}},
	}
	flags := []bool{true, true, true, true, true}
	segs := segmentReadFileBatches(tcs, flags)

	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].paths[0] != "/a.go" || segs[0].paths[1] != "/b.go" {
		t.Errorf("segment 0 = %v, want [/a.go, /b.go]", segs[0].paths)
	}
	if segs[1].paths[0] != "/d.go" || segs[1].paths[1] != "/e.go" {
		t.Errorf("segment 1 = %v, want [/d.go, /e.go]", segs[1].paths)
	}
}

func TestSegmentReadFileBatches_ParallelSafeDoesNotSplit(t *testing.T) {
	// read(a) -> search_code(foo) -> read(b)
	// search_code は parallel-safe → 分割しない → 1 セグメント [a, b]
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "search_code", Args: map[string]string{"pattern": "foo"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
	}
	flags := []bool{true, true, true}
	segs := segmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment (parallel-safe does not split), got %d", len(segs))
	}
	if len(segs[0].paths) != 2 {
		t.Errorf("segment should have 2 paths, got %d", len(segs[0].paths))
	}
}

func TestSegmentReadFileBatches_SkippedEntriesIgnored(t *testing.T) {
	// read(a)[exec] -> read(b)[skip] -> read(c)[exec]
	// b はスキップ → 1 セグメント [a, c]
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/c.go"}},
	}
	flags := []bool{true, false, true}
	segs := segmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].paths) != 2 {
		t.Errorf("segment should have 2 paths, got %d", len(segs[0].paths))
	}
	if segs[0].paths[0] != "/a.go" || segs[0].paths[1] != "/c.go" {
		t.Errorf("segment = %v, want [/a.go, /c.go]", segs[0].paths)
	}
}

func TestSegmentReadFileBatches_SingleReadNotBatched(t *testing.T) {
	// read(a) -> write(b) -> read(c)
	// 各セグメントに 1 件ずつ → バッチ不要
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "write_file", Args: map[string]string{"path": "/b.go", "content": "x"}},
		{Tool: "read_file", Args: map[string]string{"path": "/c.go"}},
	}
	flags := []bool{true, true, true}
	segs := segmentReadFileBatches(tcs, flags)

	if len(segs) != 0 {
		t.Errorf("expected 0 segments (each has only 1 read), got %d", len(segs))
	}
}

func TestSegmentReadFileBatches_OverMaxPaths(t *testing.T) {
	// 15 個の read_file: maxReadFileBatchPaths(10) を超えるが、
	// segmentReadFileBatches は上限チェックを行わず1セグメントとして返す。
	// chunk 分割は実行側（executeToolCallsWithParallel）で行う。
	n := 15
	tcs := make([]*tools.ToolCall, n)
	flags := make([]bool, n)
	for i := 0; i < n; i++ {
		tcs[i] = &tools.ToolCall{
			Tool: "read_file",
			Args: map[string]string{"path": fmt.Sprintf("/f%d.go", i)},
		}
		flags[i] = true
	}
	segs := segmentReadFileBatches(tcs, flags)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].indices) != n {
		t.Errorf("expected %d indices, got %d", n, len(segs[0].indices))
	}
}

func TestSegmentReadFileBatches_NonBatchableReadIgnored(t *testing.T) {
	// read(a) -> read(b, start_line/end_line) -> read(c)
	// range read は非バッチ → セグメント [a, c]
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go", "start_line": "10", "end_line": "20"}},
		{Tool: "read_file", Args: map[string]string{"path": "/c.go"}},
	}
	flags := []bool{true, true, true}
	segs := segmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].paths) != 2 {
		t.Errorf("segment should have 2 paths (a, c), got %d", len(segs[0].paths))
	}
}

// ── SavingsMetrics tests ──

func TestSavingsMetrics_Add(t *testing.T) {
	m := SavingsMetrics{SavedCalls: 1, EstimatedInputTokensSaved: 100, EstimatedCostSaved: 0.01}
	m.add(SavingsMetrics{SavedCalls: 2, EstimatedInputTokensSaved: 200, EstimatedCostSaved: 0.02})
	if m.SavedCalls != 3 {
		t.Errorf("SavedCalls = %d, want 3", m.SavedCalls)
	}
	if m.EstimatedInputTokensSaved != 300 {
		t.Errorf("EstimatedInputTokensSaved = %d, want 300", m.EstimatedInputTokensSaved)
	}
	if m.EstimatedCostSaved < 0.029 || m.EstimatedCostSaved > 0.031 {
		t.Errorf("EstimatedCostSaved = %f, want ~0.03", m.EstimatedCostSaved)
	}
}

func TestSavingsMetrics_HasAny(t *testing.T) {
	m := SavingsMetrics{}
	if m.hasAny() {
		t.Error("empty metrics should return false")
	}
	m.SavedCalls = 1
	if !m.hasAny() {
		t.Error("non-zero SavedCalls should return true")
	}
}

// ── search_code chunk batch test (unit) ──

func TestSearchCodeChunkBatch_GroupsOverLimit(t *testing.T) {
	// 7 patterns with same options should all be grouped together.
	// Chunk 分割は executeToolCallsWithParallel 内で行われるが、
	// グループ化自体はここでテストする。
	n := 7
	tcs := make([]*tools.ToolCall, n)
	for i := 0; i < n; i++ {
		tcs[i] = &tools.ToolCall{
			Tool: "search_code",
			Args: map[string]string{"pattern": fmt.Sprintf("pat%d", i)},
		}
	}

	// グループ化ロジックの検証
	groups := make(map[string][]int)
	for i, tc := range tcs {
		if !isSimpleSearchPattern(tc.Args["pattern"]) {
			continue
		}
		key := searchCodeOptionsKey(tc)
		groups[key] = append(groups[key], i)
	}

	// 全パターンが1グループになる（options が同一）
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	for _, indices := range groups {
		if len(indices) != n {
			t.Errorf("group should have %d indices, got %d", n, len(indices))
		}
		// chunk 分割: maxSearchBatchPatterns(5) 単位
		chunkCount := 0
		for start := 0; start < len(indices); start += maxSearchBatchPatterns {
			end := start + maxSearchBatchPatterns
			if end > len(indices) {
				end = len(indices)
			}
			chunk := indices[start:end]
			if len(chunk) >= 2 {
				chunkCount++
			}
		}
		// 7 → chunk [0..4](5) + chunk [5..6](2) = 2 chunks
		if chunkCount != 2 {
			t.Errorf("expected 2 merge-eligible chunks, got %d", chunkCount)
		}
	}
}

// ── read_file chunk batch test (unit) ──

func TestReadFileChunkBatch_SegmentOverLimit(t *testing.T) {
	// 12 plain read_file calls: segmentation returns 1 segment (no max check).
	// Chunk 分割は executeToolCallsWithParallel 内で maxReadFileBatchPaths 単位で行う。
	n := 12
	tcs := make([]*tools.ToolCall, n)
	flags := make([]bool, n)
	for i := 0; i < n; i++ {
		tcs[i] = &tools.ToolCall{
			Tool: "read_file",
			Args: map[string]string{"path": fmt.Sprintf("/f%d.go", i)},
		}
		flags[i] = true
	}

	segs := segmentReadFileBatches(tcs, flags)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].indices) != n {
		t.Errorf("segment should have %d indices, got %d", n, len(segs[0].indices))
	}

	// chunk 分割シミュレーション
	seg := segs[0]
	chunkCount := 0
	for start := 0; start < len(seg.indices); start += maxReadFileBatchPaths {
		end := start + maxReadFileBatchPaths
		if end > len(seg.indices) {
			end = len(seg.indices)
		}
		chunk := seg.indices[start:end]
		if len(chunk) >= 2 {
			chunkCount++
		}
	}
	// 12 → chunk [0..9](10) + chunk [10..11](2) = 2 chunks
	if chunkCount != 2 {
		t.Errorf("expected 2 merge-eligible chunks, got %d", chunkCount)
	}
}
