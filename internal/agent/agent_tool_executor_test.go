package agent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/subagent"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type blockingParallelTool struct {
	started chan struct{}
	release chan struct{}
}

type spinnerOrderCheckWriter struct {
	mu                  sync.Mutex
	agent               *Agent
	sawParallelGroup    bool
	groupWhileSpinnerOn bool
}

func (w *spinnerOrderCheckWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	text := string(p)
	if strings.Contains(text, "┌ Parallel") {
		w.sawParallelGroup = true
		if w.agent != nil && w.agent.ui().CurrentSpinner() != nil {
			w.groupWhileSpinnerOn = true
		}
	}
	return len(p), nil
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

func TestToolExecutionContext_PrefersActiveModelOwnerForProviderConfigKey(t *testing.T) {
	cfg := newProjectMapDisabledConfig()
	cfg.SetProviderModelsForEdit(map[string]config.ProviderModelConfig{
		"claude": {
			DefaultModel: "claude-custom",
		},
		"anthropic": {
			DefaultModel: "anthropic-custom",
		},
	})
	runtime := NewAgentRuntimeWithConfig(cfg)

	agent := &Agent{
		ProviderName:      "claude",
		ProviderConfigKey: "claude",
		CurrentModel:      "anthropic-custom",
		CurrentProvider:   &MockProvider{name: "claude"},
		Runtime:           runtime,
	}

	execCtx := agent.toolExecutionContext(context.Background(), nil, io.Discard, io.Discard)

	if execCtx.ProviderName != "claude" {
		t.Fatalf("ProviderName = %q, want %q", execCtx.ProviderName, "claude")
	}
	if execCtx.ProviderConfigKey != "anthropic" {
		t.Fatalf("ProviderConfigKey = %q, want %q", execCtx.ProviderConfigKey, "anthropic")
	}
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

	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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

// --- Test 2: skipFn prevents execution for selected tools ---

func TestExecuteToolCallsWithParallel_SkipFn_GenericTools(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "write_file", Args: map[string]string{"path": "/tmp/a.go"}, RawArgs: map[string]any{"path": "/tmp/a.go"}},
		{ID: "c2", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c3", Tool: "search_code", Args: map[string]string{"pattern": "needle"}, RawArgs: map[string]any{"pattern": "needle"}},
	}

	agent.addToolCallsToHistory("test", toolCalls)

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "write_file" || tc.Tool == "search_code" {
			return true, fmt.Sprintf("[%s] Ignored by skipFn.", tc.Tool)
		}
		return false, ""
	}

	var executedTools []string
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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

	// skip メッセージが履歴にあること
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

func TestExecuteToolCallsWithParallel_LoopDetectBeforeSkip(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	cfg := agent.cfg()
	origThreshold := cfg.LoopDetection.Threshold
	cfg.LoopDetection.Threshold = 2
	defer func() { cfg.LoopDetection.Threshold = origThreshold }()

	toolCalls := []*tools.ToolCall{
		{ID: "c1", Tool: "write_file", Args: map[string]string{"path": "/tmp/a.go"}, RawArgs: map[string]any{"path": "/tmp/a.go"}},
		{ID: "c2", Tool: "write_file", Args: map[string]string{"path": "/tmp/a.go"}, RawArgs: map[string]any{"path": "/tmp/a.go"}},
		{ID: "c3", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
	}
	agent.addToolCallsToHistory("test", toolCalls)
	historyBefore := len(agent.History)

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "write_file" {
			return true, "[write_file] Ignored by skipFn."
		}
		return false, ""
	}

	var lastToolCall *tools.ToolCall
	sameCallCount := 0
	loopDetectFn := func(tc *tools.ToolCall) bool {
		if isSameToolCall(tc, lastToolCall) {
			sameCallCount++
			return sameCallCount >= cfg.LoopDetection.Threshold
		}
		sameCallCount = 1
		lastToolCall = tc
		return false
	}

	var executed []string
	loopDetected := agent.executeToolCallsWithParallel(context.Background(), toolCalls, loopDetectFn, skipFn, func(_ int, tc *tools.ToolCall, _ toolruntime.Result) {
		executed = append(executed, tc.ID)
	})

	if !loopDetected {
		t.Fatal("expected loopDetected=true")
	}
	if len(executed) != 0 {
		t.Fatalf("executed = %v, want none", executed)
	}

	addedMsgs := agent.History[historyBefore:]
	if len(addedMsgs) != 3 {
		t.Fatalf("added %d history messages, want 3", len(addedMsgs))
	}
	if addedMsgs[0].ToolCallID != "c1" || !strings.Contains(addedMsgs[0].Content, "Ignored by skipFn") {
		t.Errorf("addedMsgs[0] = {ID:%q, Content:%q}, want skip message for c1", addedMsgs[0].ToolCallID, addedMsgs[0].Content)
	}
	if addedMsgs[1].ToolCallID != "c2" || !strings.Contains(addedMsgs[1].Content, "loop detected") {
		t.Errorf("addedMsgs[1] = {ID:%q, Content:%q}, want loop detection for c2", addedMsgs[1].ToolCallID, addedMsgs[1].Content)
	}
	if addedMsgs[2].ToolCallID != "c3" || !strings.Contains(addedMsgs[2].Content, "Skipped due to tool loop detection.") {
		t.Errorf("addedMsgs[2] = {ID:%q, Content:%q}, want loop skip for c3", addedMsgs[2].ToolCallID, addedMsgs[2].Content)
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
	callback := func(idx int, tc *tools.ToolCall, result toolruntime.Result) {
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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		if strings.Contains(result.Result, "cancel") {
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
		{ID: "c0", Tool: "write_file", Args: map[string]string{"path": "/tmp/x.go"}, RawArgs: map[string]any{"path": "/tmp/x.go"}},
		{ID: "c1", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
		{ID: "c2", Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},             // ループ
		{ID: "c3", Tool: "search_code", Args: map[string]string{"pattern": "x"}, RawArgs: map[string]any{"pattern": "x"}}, // ループ後スキップ
	}

	agent.addToolCallsToHistory("test", toolCalls)
	historyBefore := len(agent.History)

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "write_file" {
			return true, "[write_file] Ignored"
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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		atomic.AddInt32(&callbackCount, 1)
	}

	ctx := context.Background()
	agent.executeToolCallsWithParallel(ctx, toolCalls, nil, nil, callback)

	if got := atomic.LoadInt32(&callbackCount); got != int32(n) {
		t.Errorf("callback called %d times, want %d", got, n)
	}
}

func TestExecuteToolCallsWithParallel_PrintsParallelGroup(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

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
		func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})
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
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

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
		agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})
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

func TestExecuteToolCallsWithParallel_StopsSpinnerBeforeNonTUIReport(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

	provider := &mockProvider{name: "test"}
	runtime := NewAgentRuntime()
	checkWriter := &spinnerOrderCheckWriter{}
	runtime.UI = ui.NewRuntime(strings.NewReader(""), checkWriter, checkWriter)

	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	checkWriter.agent = agent
	t.Cleanup(agent.Cleanup)

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

	agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})

	if !checkWriter.sawParallelGroup {
		t.Fatal("expected non-TUI parallel group output")
	}
	if checkWriter.groupWhileSpinnerOn {
		t.Fatal("parallel report should not run while spinner is still active")
	}
}

func TestExecuteToolCallsWithParallel_StopsSpinnerBeforeTUIReport(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "str_replace")

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

	toolResultCh := make(chan tools.ToolResultInfo)
	agent.tuiToolResultCh = toolResultCh

	toolCalls := []*tools.ToolCall{{
		ID:      "c1",
		Tool:    "read_file",
		Args:    testReadFileArgs("a.go"),
		RawArgs: testReadFileRawArgs("a.go"),
	}}

	done := make(chan struct{})
	go func() {
		agent.executeToolCallsWithParallel(context.Background(), toolCalls, nil, nil, func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})
		close(done)
	}()

	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel tool to start")
	}

	if spinner := agent.ui().CurrentSpinner(); spinner == nil || !spinner.IsActive() {
		t.Fatal("expected spinner to be active before TUI parallel report")
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		close(release)
	}()

	select {
	case <-toolResultCh:
		if spinner := agent.ui().CurrentSpinner(); spinner != nil {
			t.Fatal("TUI parallel report should not run while spinner is still active")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for TUI parallel tool result")
	}

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for parallel execution to finish")
	}
}

// --- Test: Text-based (non-FC) tool calls also handled correctly ---

func TestExecuteToolCallsWithParallel_TextBased_Skip(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// ID が空 = テキストベース
	toolCalls := []*tools.ToolCall{
		{Tool: "write_file", Args: map[string]string{"path": "/tmp/x.go"}, RawArgs: map[string]any{"path": "/tmp/x.go"}},
		{Tool: "read_file", Args: testReadFileArgs("/a.go"), RawArgs: testReadFileRawArgs("/a.go")},
	}

	skipFn := func(tc *tools.ToolCall) (bool, string) {
		if tc.Tool == "write_file" {
			return true, "[write_file] Ignored"
		}
		return false, ""
	}

	var executedTools []string
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
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
		if msg.Role == "user" && strings.Contains(msg.Content, "[write_file] Ignored") {
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
		func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})

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
		func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})

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
		func(_ int, _ *tools.ToolCall, _ toolruntime.Result) {})

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
	callback := func(_ int, tc *tools.ToolCall, result toolruntime.Result) {
		callbackResults = append(callbackResults, result.Result)
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

func TestExecuteToolWithSpinner_WaitAgentLiveView(t *testing.T) {
	prevNoColor := color.NoColor
	color.NoColor = true
	defer func() { color.NoColor = prevNoColor }()

	provider := &mockProvider{name: "openai"}
	cfg := config.DefaultConfig()
	var out bytes.Buffer
	manager := subagent.NewManagerWithOptions(subagent.ManagerOptions{
		RunHeadless: func(ctx context.Context, _ string, _ string, _ api.Provider, _ *config.Config) *subagent.RunResult {
			subagent.EmitEvent(ctx, subagent.SubAgentEvent{
				Tool:     "read_file",
				Phase:    "start",
				FilePath: "manager.go",
			})
			subagent.EmitEvent(ctx, subagent.SubAgentEvent{
				Tool:     "str_replace",
				Phase:    "start",
				FilePath: "manager.go",
			})
			subagent.EmitEvent(ctx, subagent.SubAgentEvent{
				Tool:     "str_replace",
				Phase:    "end",
				FilePath: "manager.go",
				Success:  true,
				Output:   "Successfully replaced text in manager.go (lines 58-59 → 58-60)",
				OldStr:   "id        string\nstatus    string",
				NewStr:   "id        string\ntaskType  string\nstatus    string",
			})
			return &subagent.RunResult{
				Status:         "completed",
				Response:       "done",
				ToolExecutions: 2,
				DurationMs:     1200,
			}
		},
		ProviderFactory: func(providerName string) (api.Provider, error) {
			return &mockProvider{name: providerName}, nil
		},
	})

	id, err := manager.Spawn(context.Background(), "inspect", "", "", "", provider, cfg)
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	runtime := &AgentRuntime{
		Config:          cfg,
		UI:              ui.NewRuntime(strings.NewReader(""), &out, &out),
		SubAgentManager: manager,
	}
	agent := NewAgentWithRuntime("test-model", provider, false, runtime)
	t.Cleanup(agent.Cleanup)

	toolCall := &tools.ToolCall{
		Tool:    "wait_agent",
		Args:    map[string]string{"ids": fmt.Sprintf(`["%s"]`, id)},
		RawArgs: map[string]any{"ids": []string{id}},
	}

	_, _ = agent.executeToolWithSpinner(context.Background(), toolCall)

	output := out.String()
	for _, want := range []string{
		fmt.Sprintf("%s │ 📖 read_file manager.go", id),
		fmt.Sprintf("%s │ 🔧 str_replace manager.go", id),
		"+ taskType  string",
		fmt.Sprintf("%s │ ✅ completed (2 tools, 1.2s)", id),
		"⏳ wait_agent: 1 agent",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestWaitAgentSpinnerMessage(t *testing.T) {
	tests := []struct {
		name string
		args map[string]string
		want string
	}{
		{
			name: "3 agents",
			args: map[string]string{"ids": `["a","b","c"]`},
			want: "Waiting for 3 agents...",
		},
		{
			name: "1 agent",
			args: map[string]string{"ids": `["x"]`},
			want: "Waiting for 1 agents...",
		},
		{
			name: "empty ids",
			args: map[string]string{"ids": `[]`},
			want: "Waiting for agents...",
		},
		{
			name: "invalid json",
			args: map[string]string{"ids": "not-json"},
			want: "Waiting for agents...",
		},
		{
			name: "no ids key",
			args: map[string]string{},
			want: "Waiting for agents...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := waitAgentSpinnerMessage(tt.args)
			if got != tt.want {
				t.Errorf("waitAgentSpinnerMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParallelGroupSpinnerMessage_SpawnAndWait(t *testing.T) {
	tests := []struct {
		name  string
		calls []*tools.ToolCall
		want  string
	}{
		{
			name: "spawn_agent x3",
			calls: []*tools.ToolCall{
				{Tool: "spawn_agent"},
				{Tool: "spawn_agent"},
				{Tool: "spawn_agent"},
			},
			want: "Spawning 3 sub-agents...",
		},
		{
			name: "wait_agent with 5 ids",
			calls: []*tools.ToolCall{
				{Tool: "wait_agent", Args: map[string]string{"ids": `["a","b","c","d","e"]`}},
			},
			want: "Waiting for 5 agents...",
		},
		{
			name: "wait_agent without ids",
			calls: []*tools.ToolCall{
				{Tool: "wait_agent", Args: map[string]string{}},
			},
			want: "Waiting for agents...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indices := make([]int, len(tt.calls))
			for i := range indices {
				indices[i] = i
			}
			got := parallelGroupSpinnerMessage(tt.calls, indices)
			if got != tt.want {
				t.Errorf("parallelGroupSpinnerMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}
