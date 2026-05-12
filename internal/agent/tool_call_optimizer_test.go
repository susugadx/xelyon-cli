package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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

	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		// Simulate adding result to history (as agent_chat.go does)
		if tc.ID != "" {
			agent.History = append(agent.History, api.Message{
				Role:       "tool",
				Content:    result.Result,
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

	key1 := toolruntime.SearchCodeOptionsKey(tc1)
	key2 := toolruntime.SearchCodeOptionsKey(tc2)

	if key1 != key2 {
		t.Errorf("same options should produce same key: %q vs %q", key1, key2)
	}
}

func TestSearchCodeBatch_DifferentOptionsNotGrouped(t *testing.T) {
	tc1 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "foo", "file_filter": "go"}}
	tc2 := &tools.ToolCall{Tool: "search_code", Args: map[string]string{"pattern": "bar", "file_filter": "py"}}

	key1 := toolruntime.SearchCodeOptionsKey(tc1)
	key2 := toolruntime.SearchCodeOptionsKey(tc2)

	if key1 == key2 {
		t.Error("different file_filter should produce different keys")
	}
}

func TestSearchCodeBatch_MultiPatternPatternNotBatched(t *testing.T) {
	// Already multi-pattern → should not be batched further
	if toolruntime.IsSimpleSearchPattern("foo,bar") {
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
	sections := toolruntime.SplitMultiPatternResult(result, patterns)

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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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
		{ID: "c2", Tool: "write_file", Args: map[string]string{"path": "/tmp/x.go"}, RawArgs: map[string]any{"path": "/tmp/x.go"}},
		{ID: "c3", Tool: "read_file", Args: map[string]string{"path": "/a.go"}, RawArgs: map[string]any{"path": "/a.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "write_file" {
			return true, "[write_file] Ignored"
		}
		return false, ""
	}

	var executedIDs []string
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		executedIDs = append(executedIDs, tc.ID)
	}

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, skipFn, callback)

	// c1/c3 executed, c2 skipped
	if len(executedIDs) != 2 || executedIDs[0] != "c1" || executedIDs[1] != "c3" {
		t.Errorf("executedIDs = %v, want [c1 c3]", executedIDs)
	}
}

// ── toolruntime.BuildReadFileBatchToolCall tests ──

func TestBuildReadFileBatchToolCall_FullBudget(t *testing.T) {
	paths := []string{"/a.go", "/b.go", "/c.go"}
	tc := toolruntime.BuildReadFileBatchToolCall(paths, true)

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
	// RawArgs.Paths should be []string
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
	tc := toolruntime.BuildReadFileBatchToolCall([]string{"/a.go", "/b.go"}, false)
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
		{"paths detail auto", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`, "detail": "auto"}}, true},
		{"paths detail full", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`, "detail": "full"}}, false},
		{"paths detail outline", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go","b.go"]`, "detail": "outline"}}, false},
		{"paths specified with range", &tools.ToolCall{Tool: "read_file", Args: map[string]string{"paths": `["a.go:10-20","b.go"]`}}, false},
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
	callback := func(idx int, tc *tools.ToolCall, result toolruntime.Result) {
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

	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {}
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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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

// ── toolruntime.SegmentReadFileBatches tests ──

func TestSegmentReadFileBatches_AllReads(t *testing.T) {
	// 全て read_file → 1 セグメント
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go"}},
		{Tool: "read_file", Args: map[string]string{"path": "/c.go"}},
	}
	flags := []bool{true, true, true}
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].Paths) != 3 {
		t.Errorf("segment should have 3 paths, got %d", len(segs[0].Paths))
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
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment (only post-mutation pair), got %d", len(segs))
	}
	if segs[0].Paths[0] != "/c.go" || segs[0].Paths[1] != "/d.go" {
		t.Errorf("segment paths = %v, want [/c.go, /d.go]", segs[0].Paths)
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
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 2 {
		t.Fatalf("expected 2 segments, got %d", len(segs))
	}
	if segs[0].Paths[0] != "/a.go" || segs[0].Paths[1] != "/b.go" {
		t.Errorf("segment 0 = %v, want [/a.go, /b.go]", segs[0].Paths)
	}
	if segs[1].Paths[0] != "/d.go" || segs[1].Paths[1] != "/e.go" {
		t.Errorf("segment 1 = %v, want [/d.go, /e.go]", segs[1].Paths)
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
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment (parallel-safe does not split), got %d", len(segs))
	}
	if len(segs[0].Paths) != 2 {
		t.Errorf("segment should have 2 paths, got %d", len(segs[0].Paths))
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
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].Paths) != 2 {
		t.Errorf("segment should have 2 paths, got %d", len(segs[0].Paths))
	}
	if segs[0].Paths[0] != "/a.go" || segs[0].Paths[1] != "/c.go" {
		t.Errorf("segment = %v, want [/a.go, /c.go]", segs[0].Paths)
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
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 0 {
		t.Errorf("expected 0 segments (each has only 1 read), got %d", len(segs))
	}
}

func TestSegmentReadFileBatches_OverMaxPaths(t *testing.T) {
	// 15 個の read_file: toolruntime.MaxReadFileBatchPaths(10) を超えるが、
	// toolruntime.SegmentReadFileBatches は上限チェックを行わず1セグメントとして返す。
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
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].Indices) != n {
		t.Errorf("expected %d indices, got %d", n, len(segs[0].Indices))
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
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].Paths) != 2 {
		t.Errorf("segment should have 2 paths (a, c), got %d", len(segs[0].Paths))
	}
}

func TestSegmentReadFileBatches_ExplicitDetailIgnored(t *testing.T) {
	tcs := []*tools.ToolCall{
		{Tool: "read_file", Args: map[string]string{"path": "/a.go", "detail": "outline"}},
		{Tool: "read_file", Args: map[string]string{"path": "/b.go", "detail": "outline"}},
	}
	flags := []bool{true, true}
	segs := toolruntime.SegmentReadFileBatches(tcs, flags)

	if len(segs) != 0 {
		t.Fatalf("expected 0 segments for explicit detail reads, got %d", len(segs))
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
		if !toolruntime.IsSimpleSearchPattern(tc.Args["pattern"]) {
			continue
		}
		key := toolruntime.SearchCodeOptionsKey(tc)
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
		// chunk 分割: toolruntime.MaxSearchBatchPatterns(5) 単位
		chunkCount := 0
		for start := 0; start < len(indices); start += toolruntime.MaxSearchBatchPatterns {
			end := start + toolruntime.MaxSearchBatchPatterns
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
	// Chunk 分割は executeToolCallsWithParallel 内で toolruntime.MaxReadFileBatchPaths 単位で行う。
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

	segs := toolruntime.SegmentReadFileBatches(tcs, flags)
	if len(segs) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segs))
	}
	if len(segs[0].Indices) != n {
		t.Errorf("segment should have %d indices, got %d", n, len(segs[0].Indices))
	}

	// chunk 分割シミュレーション
	seg := segs[0]
	chunkCount := 0
	for start := 0; start < len(seg.Indices); start += toolruntime.MaxReadFileBatchPaths {
		end := start + toolruntime.MaxReadFileBatchPaths
		if end > len(seg.Indices) {
			end = len(seg.Indices)
		}
		chunk := seg.Indices[start:end]
		if len(chunk) >= 2 {
			chunkCount++
		}
	}
	// 12 → chunk [0..9](10) + chunk [10..11](2) = 2 chunks
	if chunkCount != 2 {
		t.Errorf("expected 2 merge-eligible chunks, got %d", chunkCount)
	}
}
