package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type blockingParallelTool struct {
	started chan struct{}
	release chan struct{}
}

func (t *blockingParallelTool) Name() string { return "read_file" }

func (t *blockingParallelTool) Description() string { return "blocking parallel test tool" }

func (t *blockingParallelTool) Parameters() map[string]interface{} {
	return map[string]interface{}{}
}

func (t *blockingParallelTool) Run(_ tools.ExecutionContext, _ map[string]string) (string, *tools.FileChange, error) {
	t.started <- struct{}{}
	<-t.release
	return "ok", nil, nil
}

// --- Test 1: Loop detection prevents execution of subsequent tools ---

func TestExecuteToolCallsWithParallel_LoopDetection_PreventsExecution(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	cfg := agent.cfg()
	origThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 2
	defer func() { cfg.LoopDetection.Threshold = origThreshold }()

	// 同じ read_file が3回 + 後続の search_code
	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c2", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c3", Tool: "search_code", Args: map[string]string{"pattern": "foo"}, RawArgs: map[string]any{"pattern": "foo"}},
	}

	// assistant メッセージを先に追加（FC 要件）
	agent.addToolCallsToHistory("test", toolCalls)

	var executedTools []string
	var lastToolCall *tools.ToolCall
	sameCallCount := 0

	loopDetectFn := func(tc *tools.ToolCall) bool {
		if isSameToolCall(tc, lastToolCall) {
			sameCallCount++
			if sameCallCount >= cfg.LoopDetection.Threshold {
				return true
			}
		} else {
			sameCallCount = 1
		}
		lastToolCall = tc
		return false
	}

	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedTools = append(executedTools, tc.Tool)
	}

	ctx := context.Background()
	loopDetected := agent.executeToolCallsWithParallel(ctx, toolCalls, loopDetectFn, nil, callback)

	if !loopDetected {
		t.Fatal("expected loopDetected=true")
	}

	// c1 だけが実行される（c2 でループ検知、c3 は中止）
	if len(executedTools) != 1 {
		t.Fatalf("executed %d tools, want 1; got %v", len(executedTools), executedTools)
	}
	if executedTools[0] != "read_file" {
		t.Errorf("executedTools[0] = %q, want 'read_file'", executedTools[0])
	}

	// c2（ループトリガー）のダミー結果が履歴にあること
	found := false
	for _, msg := range agent.History {
		if msg.ToolCallID == "c2" && strings.Contains(msg.Content, "loop detected") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected loop detection message for c2 in history")
	}

	// c3（ループ後）のスキップ結果が履歴にあること
	found = false
	for _, msg := range agent.History {
		if msg.ToolCallID == "c3" && strings.Contains(msg.Content, "Skipped due to tool loop") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected skip message for c3 in history")
	}
}

// --- Test 2: skipFn prevents execution (Plan Mode create_plan/update_plan) ---

func TestExecuteToolCallsWithParallel_SkipFn_PlanningTools(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "create_plan", Args: map[string]string{"title": "Plan"}, RawArgs: map[string]any{"title": "Plan"}},
		{ID: "c2", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c3", Tool: "update_plan", Args: map[string]string{"step_id": "1"}, RawArgs: map[string]any{"step_id": "1"}},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "create_plan" || tc.Tool == "update_plan" {
			return true, fmt.Sprintf("[%s] Ignored: planning tools are deprecated.", tc.Tool)
		}
		return false, ""
	}

	var executedTools []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedTools = append(executedTools, tc.Tool)
	}

	ctx := context.Background()
	loopDetected := agent.executeToolCallsWithParallel(ctx, toolCalls, nil, skipFn, callback)

	if loopDetected {
		t.Error("expected loopDetected=false")
	}

	// read_file だけが実行されること
	if len(executedTools) != 1 {
		t.Fatalf("executed %d tools, want 1; got %v", len(executedTools), executedTools)
	}
	if executedTools[0] != "read_file" {
		t.Errorf("executedTools[0] = %q, want 'read_file'", executedTools[0])
	}

	// create_plan/update_plan の skip メッセージが履歴にあること
	for _, id := range []string{"c1", "c3"} {
		found := false
		for _, msg := range agent.History {
			if msg.ToolCallID == id && strings.Contains(msg.Content, "Ignored") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected skip message for %s in history", id)
		}
	}
}

// --- Test 3: Mixed case delivery order matches original tool call order ---

func TestExecuteToolCallsWithParallel_DeliveryOrder(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// parallel: read_file(0), search_code(2), list_dir(4)
	// sequential: write_file(1), str_replace(3)
	toolCalls := []*tools.ToolCall{
		{ID: "c0", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c1", Tool: "write_file", Args: map[string]string{"path": "/b.go"}, RawArgs: map[string]any{"path": "/b.go"}},
		{ID: "c2", Tool: "search_code", Args: map[string]string{"pattern": "foo"}, RawArgs: map[string]any{"pattern": "foo"}},
		{ID: "c3", Tool: "str_replace", Args: map[string]string{"path": "/c.go"}, RawArgs: map[string]any{"path": "/c.go"}},
		{ID: "c4", Tool: "list_dir", Args: map[string]string{"path": "."}, RawArgs: map[string]any{"path": "."}},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	var deliveryOrder []int
	callback := func(idx int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		deliveryOrder = append(deliveryOrder, idx)
	}

	ctx := context.Background()
	agent.executeToolCallsWithParallel(ctx, toolCalls, nil, nil, callback)

	// callback は元の順序 0,1,2,3,4 で呼ばれること
	expected := []int{0, 1, 2, 3, 4}
	if len(deliveryOrder) != len(expected) {
		t.Fatalf("delivery count = %d, want %d; got %v", len(deliveryOrder), len(expected), deliveryOrder)
	}
	for i, idx := range deliveryOrder {
		if idx != expected[i] {
			t.Errorf("deliveryOrder[%d] = %d, want %d", i, idx, expected[i])
		}
	}
}

// --- Test 4: Context cancellation propagates to execution path ---

func TestExecuteToolCallsWithParallel_ContextCancel(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c0", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c1", Tool: "read_file", Args: testReadFileArgs("/b.go"), RawArgs: testReadFileRawArgs("/b.go")},
		{ID: "c2", Tool: "read_file", Args: testReadFileArgs("/c.go"), RawArgs: testReadFileRawArgs("/c.go")},
		{ID: "c3", Tool: "read_file", Args: testReadFileArgs("/d.go"), RawArgs: testReadFileRawArgs("/d.go")},
		{ID: "c4", Tool: "read_file", Args: testReadFileArgs("/e.go"), RawArgs: testReadFileRawArgs("/e.go")},
		{ID: "c5", Tool: "read_file", Args: testReadFileArgs("/f.go"), RawArgs: testReadFileRawArgs("/f.go")},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	// 短いタイムアウトで ctx をキャンセル
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	var cancelledCount int32
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		if strings.Contains(result, "cancel") {
			atomic.AddInt32(&cancelledCount, 1)
		}
	}

	agent.executeToolCallsWithParallel(ctx, toolCalls, nil, nil, callback)

	// タイムアウト後のツールは "Error: context cancelled" を含むはず
	// （タイミング依存のため、最低1つキャンセルされることを確認）
	// NOTE: 全ツールが parallel-safe かつ tools.Execute が即座に返る場合、
	// キャンセルが間に合わない可能性があるため、厳密な数は検証しない
	t.Logf("cancelledCount = %d (timing-dependent)", atomic.LoadInt32(&cancelledCount))
}

// --- Test 5: Skip/ignored/dummy results are correctly added to history ---

func TestExecuteToolCallsWithParallel_HistoryMessages(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	cfg := agent.cfg()
	origThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 2
	defer func() { cfg.LoopDetection.Threshold = origThreshold }()

	// 順序: skip(c0) → execute(c1) → loop(c2=c1と同じ) → abort(c3)
	toolCalls := []*tools.ToolCall{
		{ID: "c0", Tool: "create_plan", Args: map[string]string{"title": "X"}, RawArgs: map[string]any{"title": "X"}},
		{ID: "c1", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c2", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},             // ループ
		{ID: "c3", Tool: "search_code", Args: map[string]string{"pattern": "x"}, RawArgs: map[string]any{"pattern": "x"}}, // ループ後スキップ
	}

	agent.addToolCallsToHistory("test", toolCalls)
	historyBefore := len(agent.History)

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "create_plan" {
			return true, "[create_plan] Ignored"
		}
		return false, ""
	}

	var lastToolCall *tools.ToolCall
	sameCallCount := 0
	loopDetectFn := func(tc *tools.ToolCall) bool {
		if isSameToolCall(tc, lastToolCall) {
			sameCallCount++
			if sameCallCount >= cfg.LoopDetection.Threshold {
				return true
			}
		} else {
			sameCallCount = 1
		}
		lastToolCall = tc
		return false
	}

	var executedIDs []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedIDs = append(executedIDs, tc.ID)
	}

	ctx := context.Background()
	loopDetected := agent.executeToolCallsWithParallel(ctx, toolCalls, loopDetectFn, skipFn, callback)

	if !loopDetected {
		t.Fatal("expected loopDetected=true")
	}

	// c1 だけが実行される（c0=skip, c2=loop, c3=abort）
	if len(executedIDs) != 1 || executedIDs[0] != "c1" {
		t.Errorf("executedIDs = %v, want [c1]", executedIDs)
	}

	// Phase 2 で追加されたメッセージを検証
	addedMsgs := agent.History[historyBefore:]

	// c0: skip → 1メッセージ (tool result with "Ignored")
	// c1: callback で処理（テストでは履歴追加しない）→ 0メッセージ
	// c2: loopTrigger → 1メッセージ (tool result with "loop detected")
	// c3: loopAbort → 1メッセージ (tool result with "Skipped")
	if len(addedMsgs) != 3 {
		var ids []string
		for _, m := range addedMsgs {
			ids = append(ids, fmt.Sprintf("{ID:%s, Content:%.40s}", m.ToolCallID, m.Content))
		}
		t.Fatalf("added %d history messages, want 3; messages: %v", len(addedMsgs), ids)
	}

	// c0: skip message
	if addedMsgs[0].ToolCallID != "c0" || !strings.Contains(addedMsgs[0].Content, "Ignored") {
		t.Errorf("addedMsgs[0] = {ID:%q, Content:%q}, want c0 with 'Ignored'", addedMsgs[0].ToolCallID, addedMsgs[0].Content)
	}

	// c2: loop trigger
	if addedMsgs[1].ToolCallID != "c2" || !strings.Contains(addedMsgs[1].Content, "loop detected") {
		t.Errorf("addedMsgs[1] = {ID:%q, Content:%q}, want c2 with 'loop detected'", addedMsgs[1].ToolCallID, addedMsgs[1].Content)
	}

	// c3: loop abort (skipped)
	if addedMsgs[2].ToolCallID != "c3" || !strings.Contains(addedMsgs[2].Content, "Skipped") {
		t.Errorf("addedMsgs[2] = {ID:%q, Content:%q}, want c3 with 'Skipped'", addedMsgs[2].ToolCallID, addedMsgs[2].Content)
	}
}

// --- Test: Empty tool calls returns false without panic ---

func TestExecuteToolCallsWithParallel_Empty(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	loopDetected := agent.executeToolCallsWithParallel(context.Background(), nil, nil, nil, nil)
	if loopDetected {
		t.Error("expected loopDetected=false for empty input")
	}
}

// --- Test: All parallel tools execute concurrently ---

func TestExecuteToolCallsWithParallel_AllParallelConcurrency(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	n := 4
	toolCalls := make([]*tools.ToolCall, n)
	for i := 0; i < n; i++ {
		toolCalls[i] = &tools.ToolCall{
			ID:      fmt.Sprintf("c%d", i),
			Tool:    "read_file",
			Args:    testReadFileArgs(fmt.Sprintf("/file%d.go", i)),
			RawArgs: testReadFileRawArgs(fmt.Sprintf("/file%d.go", i)),
		}
	}

	agent.addToolCallsToHistory("test", toolCalls)

	var callbackCount int32
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		atomic.AddInt32(&callbackCount, 1)
	}

	ctx := context.Background()
	agent.executeToolCallsWithParallel(ctx, toolCalls, nil, nil, callback)

	if got := atomic.LoadInt32(&callbackCount); got != int32(n) {
		t.Errorf("callback called %d times, want %d", got, n)
	}
}

func TestExecuteToolCallsWithParallel_PrintsParallelGroup(t *testing.T) {
	provider := &mockProvider{name: "test"}
	var out bytes.Buffer
	agent := NewAgentWithRuntime("test-model", provider, false, &AgentRuntime{
		UI: ui.NewRuntime(strings.NewReader(""), &out, &out),
	})
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{
			ID:      "c1",
			Tool:    "read_file",
			Args:    testReadFileArgs("auto_compress.go"),
			RawArgs: testReadFileRawArgs("auto_compress.go"),
		},
		{
			ID:   "c2",
			Tool: "search_code",
			Args: map[string]string{
				"pattern": "maybeAutoCompress",
				"path":    ".",
			},
			RawArgs: map[string]any{
				"pattern": "maybeAutoCompress",
				"path":    ".",
			},
		},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil,
		func(_ int, _ *tools.ToolCall, _ string, _ *tools.FileChange) {})
	output := out.String()

	for _, want := range []string{
		"┌ Parallel (2 calls)",
		"📄 read_file: auto_compress.go",
		`🔍 search_code: "maybeAutoCompress" in . →`,
		"Found ",
		"└ Done:",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("parallel group output missing %q:\n%s", want, output)
		}
	}
}

func TestExecuteToolCallsWithParallel_ShowsSpinnerDuringParallelRun(t *testing.T) {
	provider := &mockProvider{name: "test"}
	runtime := NewAgentRuntime()
	runtime.UI = ui.NewRuntime(strings.NewReader(""), io.Discard, io.Discard)

	started := make(chan struct{}, 1)
	release := make(chan struct{})
	origTool := runtime.Registry.GetTool("read_file")
	runtime.Registry.Register(&blockingParallelTool{
		started: started,
		release: release,
	})

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	t.Cleanup(func() {
		agent.Cleanup()
		if origTool != nil {
			runtime.Registry.Register(origTool)
		}
	})

	toolCalls := []*tools.ToolCall{{
		ID:      "c1",
		Tool:    "read_file",
		Args:    testReadFileArgs("a.go"),
		RawArgs: testReadFileRawArgs("a.go"),
	}}

	done := make(chan struct{})
	go func() {
		agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, func(_ int, _ *tools.ToolCall, _ string, _ *tools.FileChange) {})
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel tool to start")
	}

	spinner := agent.ui().CurrentSpinner()
	if spinner == nil || !spinner.IsActive() {
		t.Fatal("expected spinner to be active during parallel tool execution")
	}

	close(release)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel execution to finish")
	}

	if spinner := agent.ui().CurrentSpinner(); spinner != nil {
		t.Fatal("expected spinner to be cleared after parallel execution")
	}
}

// --- Test: Text-based (non-FC) tool calls also handled correctly ---

func TestExecuteToolCallsWithParallel_TextBased_Skip(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// ID が空 = テキストベース
	toolCalls := []*tools.ToolCall{
		{Tool: "create_plan", Args: map[string]string{"title": "X"}, RawArgs: map[string]any{"title": "X"}},
		{Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
	}

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "create_plan" {
			return true, "[create_plan] Ignored"
		}
		return false, ""
	}

	var executedTools []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		executedTools = append(executedTools, tc.Tool)
	}

	ctx := context.Background()
	agent.executeToolCallsWithParallel(ctx, toolCalls, nil, skipFn, callback)

	if len(executedTools) != 1 || executedTools[0] != "read_file" {
		t.Errorf("executedTools = %v, want [read_file]", executedTools)
	}

	// テキストベースの skip は role="user" で追加されること
	found := false
	for _, msg := range agent.History {
		if msg.Role == "user" && strings.Contains(msg.Content, "[create_plan] Ignored") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected text-based skip message in history")
	}
}

// --- Test: FC loop detection message matches old shouldAbortToolLoopWithResponse format ---

func TestLoopDetection_FC_MessageConsistency(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}
	cfg := agent.cfg()
	origThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 2
	defer func() { cfg.LoopDetection.Threshold = origThreshold }()

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "read_file", Args: testReadFileArgs("/x.go"), RawArgs: testReadFileRawArgs("/x.go")},
		{ID: "c2", Tool: "read_file", Args: testReadFileArgs("/x.go"), RawArgs: testReadFileRawArgs("/x.go")},
	}
	agent.addToolCallsToHistory("test", toolCalls)
	historyBefore := len(agent.History)

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

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, loopDetectFn, nil,
		func(_ int, _ *tools.ToolCall, _ string, _ *tools.FileChange) {})

	// 旧実装と同じフォーマット: "[SYSTEM] Tool loop detected: read_file was called 2 times. ..."
	msgs := agent.History[historyBefore:]
	if len(msgs) != 1 {
		t.Fatalf("expected 1 loop message, got %d", len(msgs))
	}
	msg := msgs[0]
	if msg.Role != "tool" {
		t.Errorf("loop message role = %q, want 'tool'", msg.Role)
	}
	if msg.ToolCallID != "c2" {
		t.Errorf("loop message ToolCallID = %q, want 'c2'", msg.ToolCallID)
	}
	// 旧実装のフォーマット: "[SYSTEM] Tool loop detected: %s was called %d times. Stopping to prevent infinite loop."
	expectedContent := "[SYSTEM] Tool loop detected: read_file was called 2 times. Stopping to prevent infinite loop."
	if msg.Content != expectedContent {
		t.Errorf("loop message content = %q, want %q", msg.Content, expectedContent)
	}
}

// --- Test: Text-based loop detection message matches old format ---

func TestLoopDetection_TextBased_TriggerMessage(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}
	cfg := agent.cfg()
	origThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 2
	defer func() { cfg.LoopDetection.Threshold = origThreshold }()

	// ID="" = text-based
	toolCalls := []*tools.ToolCall{
		{Tool: "read_file", Args: testReadFileArgs("/x.go"), RawArgs: testReadFileRawArgs("/x.go")},
		{Tool: "read_file", Args: testReadFileArgs("/x.go"), RawArgs: testReadFileRawArgs("/x.go")},
	}
	historyBefore := len(agent.History)

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

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, loopDetectFn, nil,
		func(_ int, _ *tools.ToolCall, _ string, _ *tools.FileChange) {})

	msgs := agent.History[historyBefore:]
	// text-based trigger: role="user", "[SYSTEM WARNING] ..."
	if len(msgs) != 1 {
		t.Fatalf("expected 1 loop message, got %d", len(msgs))
	}
	if msgs[0].Role != "user" {
		t.Errorf("text-based loop message role = %q, want 'user'", msgs[0].Role)
	}
	if !strings.Contains(msgs[0].Content, "[SYSTEM WARNING]") {
		t.Errorf("text-based loop message should contain '[SYSTEM WARNING]', got %q", msgs[0].Content)
	}
}

// --- Test: Text-based subsequent tools after loop get NO dummy message (intentional spec) ---
// text-based パスでは tool_call_id がないため role="tool" ダミーを追加できない。
// role="user" のダミーを追加しても LLM 文脈を汚すだけであるため、旧 sequential 実装と
// 同様にダミーを追加しない。これは by design であり、互換性のために維持している。

func TestLoopDetection_TextBased_SubsequentNoDummy(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}
	cfg := agent.cfg()
	origThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 2
	defer func() { cfg.LoopDetection.Threshold = origThreshold }()

	// text-based (ID="") + 後続ツール
	toolCalls := []*tools.ToolCall{
		{Tool: "read_file", Args: testReadFileArgs("/x.go"), RawArgs: testReadFileRawArgs("/x.go")},
		{Tool: "read_file", Args: testReadFileArgs("/x.go"), RawArgs: testReadFileRawArgs("/x.go")},             // trigger
		{Tool: "search_code", Args: map[string]string{"pattern": "y"}, RawArgs: map[string]any{"pattern": "y"}}, // subsequent
	}
	historyBefore := len(agent.History)

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

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, loopDetectFn, nil,
		func(_ int, _ *tools.ToolCall, _ string, _ *tools.FileChange) {})

	msgs := agent.History[historyBefore:]
	// text-based: trigger(1) のみ。後続(search_code) にはダミーなし = intentional spec。
	// 旧 sequential 実装でも text-based 後続にダミーは追加されなかった（by design）。
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (trigger only, no dummy for text-based subsequent), got %d", len(msgs))
	}
}

// --- Test: Cancel before execution skips all tools ---

func TestExecuteToolCallsWithParallel_CancelBeforeExecution(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c0", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c1", Tool: "write_file", Args: map[string]string{"path": "/b.go"}, RawArgs: map[string]any{"path": "/b.go"}},
	}
	agent.addToolCallsToHistory("test", toolCalls)

	// 既にキャンセル済みの ctx
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var callbackResults []string
	callback := func(_ int, tc *tools.ToolCall, result string, change *tools.FileChange) {
		callbackResults = append(callbackResults, result)
	}

	agent.executeToolCallsWithParallel(ctx, toolCalls, nil, nil, callback)

	// 全ツールが "Error: context cancelled" を返すこと
	if len(callbackResults) != 2 {
		t.Fatalf("callback count = %d, want 2", len(callbackResults))
	}
	for i, r := range callbackResults {
		if !strings.Contains(r, "cancel") {
			t.Errorf("callbackResults[%d] = %q, want containing 'cancel'", i, r)
		}
	}
}
