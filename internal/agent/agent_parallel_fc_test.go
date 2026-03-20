package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// TestAddToolCallsToHistory_Single は単一ツール呼び出しで assistant メッセージが1つ作られることを確認（回帰テスト）
func TestAddToolCallsToHistory_Single(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	initialLen := len(agent.History)

	toolCalls := []*tools.ToolCall{
		{
			ID:      "call_1",
			Tool:    "read_file",
			Args:    map[string]string{"path": "/a.go"},
			RawArgs: map[string]any{"path": "/a.go"},
		},
	}

	agent.addToolCallsToHistory("I'll read the file.", toolCalls)

	// assistant メッセージが1つだけ追加されること
	if len(agent.History) != initialLen+1 {
		t.Fatalf("addToolCallsToHistory(1 TC) added %d messages, want 1", len(agent.History)-initialLen)
	}

	msg := agent.History[len(agent.History)-1]
	if msg.Role != "assistant" {
		t.Errorf("message role = %q, want 'assistant'", msg.Role)
	}
	if len(msg.ToolCalls) != 1 {
		t.Fatalf("ToolCalls length = %d, want 1", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].ID != "call_1" {
		t.Errorf("ToolCalls[0].ID = %q, want 'call_1'", msg.ToolCalls[0].ID)
	}
	if msg.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("ToolCalls[0].Function.Name = %q, want 'read_file'", msg.ToolCalls[0].Function.Name)
	}

	// Stats: AssistantMessages は1回
	if agent.Stats.AssistantMessages != 1 {
		t.Errorf("AssistantMessages = %d, want 1", agent.Stats.AssistantMessages)
	}
	// ToolExecutions は addToolCallsToHistory 時点ではカウントしない
	// （Phase 2 の実行時にカウントされる）
	if agent.Stats.ToolExecutions["read_file"] != 0 {
		t.Errorf("ToolExecutions['read_file'] = %d, want 0 (not counted until execution)", agent.Stats.ToolExecutions["read_file"])
	}
}

// TestAddToolCallsToHistory_Parallel はパラレル FC（2個以上）で assistant メッセージが1つだけ作られることを確認
func TestAddToolCallsToHistory_Parallel(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	initialLen := len(agent.History)

	toolCalls := []*tools.ToolCall{
		{
			ID:      "call_1",
			Tool:    "read_file",
			Args:    map[string]string{"path": "/a.go"},
			RawArgs: map[string]any{"path": "/a.go"},
		},
		{
			ID:      "call_2",
			Tool:    "read_file",
			Args:    map[string]string{"path": "/b.go"},
			RawArgs: map[string]any{"path": "/b.go"},
		},
		{
			ID:      "call_3",
			Tool:    "bash",
			Args:    map[string]string{"command": "ls"},
			RawArgs: map[string]any{"command": "ls"},
		},
	}

	agent.addToolCallsToHistory("I'll read both files and list the directory.", toolCalls)

	// assistant メッセージが1つだけ追加されること
	if len(agent.History) != initialLen+1 {
		t.Fatalf("addToolCallsToHistory(3 TCs) added %d messages, want 1", len(agent.History)-initialLen)
	}

	msg := agent.History[len(agent.History)-1]
	if msg.Role != "assistant" {
		t.Errorf("message role = %q, want 'assistant'", msg.Role)
	}

	// 3つの ToolCalls が1つのメッセージに格納されること
	if len(msg.ToolCalls) != 3 {
		t.Fatalf("ToolCalls length = %d, want 3", len(msg.ToolCalls))
	}

	// 各 ToolCall の ID と Name を確認
	expectedIDs := []string{"call_1", "call_2", "call_3"}
	expectedNames := []string{"read_file", "read_file", "bash"}
	for i, tc := range msg.ToolCalls {
		if tc.ID != expectedIDs[i] {
			t.Errorf("ToolCalls[%d].ID = %q, want %q", i, tc.ID, expectedIDs[i])
		}
		if tc.Function.Name != expectedNames[i] {
			t.Errorf("ToolCalls[%d].Function.Name = %q, want %q", i, tc.Function.Name, expectedNames[i])
		}
	}

	// Stats: AssistantMessages は1回（3回ではない）
	if agent.Stats.AssistantMessages != 1 {
		t.Errorf("AssistantMessages = %d, want 1", agent.Stats.AssistantMessages)
	}

	// ToolExecutions は addToolCallsToHistory 時点ではカウントしない
	// （Phase 2 の実行時にカウントされる）
	if agent.Stats.ToolExecutions["read_file"] != 0 {
		t.Errorf("ToolExecutions['read_file'] = %d, want 0 (not counted until execution)", agent.Stats.ToolExecutions["read_file"])
	}
	if agent.Stats.ToolExecutions["bash"] != 0 {
		t.Errorf("ToolExecutions['bash'] = %d, want 0 (not counted until execution)", agent.Stats.ToolExecutions["bash"])
	}
}

// TestAddToolCallsToHistory_ThoughtSignatureFirstOnly は ThoughtSignature が最初の TC のみに設定されることを確認
func TestAddToolCallsToHistory_ThoughtSignatureFirstOnly(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{
			ID:               "call_1",
			Tool:             "read_file",
			RawArgs:          map[string]any{"path": "/a.go"},
			ThoughtSignature: "sig-encrypted-abc",
			ThoughtParts: []map[string]any{
				{"text": "thinking...", "thought": true},
			},
		},
		{
			ID:               "call_2",
			Tool:             "bash",
			RawArgs:          map[string]any{"command": "ls"},
			ThoughtSignature: "sig-encrypted-abc", // 入力では両方にあるが、出力では最初のみ
		},
	}

	agent.addToolCallsToHistory("response", toolCalls)

	msg := agent.History[len(agent.History)-1]
	if len(msg.ToolCalls) != 2 {
		t.Fatalf("ToolCalls length = %d, want 2", len(msg.ToolCalls))
	}

	// 最初の TC に ThoughtSignature があること
	if msg.ToolCalls[0].ThoughtSignature != "sig-encrypted-abc" {
		t.Errorf("ToolCalls[0].ThoughtSignature = %q, want 'sig-encrypted-abc'", msg.ToolCalls[0].ThoughtSignature)
	}
	if len(msg.ToolCalls[0].ThoughtParts) != 1 {
		t.Errorf("ToolCalls[0].ThoughtParts length = %d, want 1", len(msg.ToolCalls[0].ThoughtParts))
	}

	// 2番目の TC に ThoughtSignature がないこと
	if msg.ToolCalls[1].ThoughtSignature != "" {
		t.Errorf("ToolCalls[1].ThoughtSignature = %q, want empty", msg.ToolCalls[1].ThoughtSignature)
	}
	if len(msg.ToolCalls[1].ThoughtParts) != 0 {
		t.Errorf("ToolCalls[1].ThoughtParts length = %d, want 0", len(msg.ToolCalls[1].ThoughtParts))
	}
}

// TestAddToolCallsToHistory_Empty は空のツール呼び出しで何も追加されないことを確認
func TestAddToolCallsToHistory_Empty(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	initialLen := len(agent.History)

	agent.addToolCallsToHistory("response", nil)

	if len(agent.History) != initialLen {
		t.Errorf("addToolCallsToHistory(nil) added %d messages, want 0", len(agent.History)-initialLen)
	}
}

// TestExecuteToolOnly_AddsToolResult は executeToolOnly がツール結果のみ追加し assistant メッセージを追加しないことを確認
func TestExecuteToolOnly_AddsToolResult(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	// まず assistant メッセージを追加
	agent.History = append(agent.History, api.Message{
		Role: "assistant",
		ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Type: "function", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"paths":["/nonexistent.txt"]}`}},
		},
	})
	historyLenBefore := len(agent.History)

	tc := &tools.ToolCall{
		ID:   "call_1",
		Tool: "read_file",
		Args: testReadFileArgs("/nonexistent.txt"),
	}

	agent.executeToolOnly(tc)

	// tool result メッセージが1つだけ追加されること（assistant は追加されない）
	added := len(agent.History) - historyLenBefore
	if added != 1 {
		t.Fatalf("executeToolOnly() added %d messages, want 1 (tool result only)", added)
	}

	lastMsg := agent.History[len(agent.History)-1]
	if lastMsg.Role != "tool" {
		t.Errorf("last message role = %q, want 'tool'", lastMsg.Role)
	}
	if lastMsg.ToolCallID != "call_1" {
		t.Errorf("last message ToolCallID = %q, want 'call_1'", lastMsg.ToolCallID)
	}
}

// TestParallelFC_HistoryStructure はパラレル FC 全体の履歴構造が正しいことを確認
// Expected: assistant[TC1,TC2] → tool(result1) → tool(result2)
func TestParallelFC_HistoryStructure(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)
	agent.Stats = &SessionStats{ToolExecutions: make(map[string]int)}

	toolCalls := []*tools.ToolCall{
		{
			ID:      "call_1",
			Tool:    "read_file",
			Args:    testReadFileArgs("/nonexistent1.txt"),
			RawArgs: testReadFileRawArgs("/nonexistent1.txt"),
		},
		{
			ID:      "call_2",
			Tool:    "read_file",
			Args:    testReadFileArgs("/nonexistent2.txt"),
			RawArgs: testReadFileRawArgs("/nonexistent2.txt"),
		},
	}

	initialLen := len(agent.History)

	// Phase 1: バッチで assistant メッセージを追加
	agent.addToolCallsToHistory("I'll read both files.", toolCalls)

	// Phase 2: 各ツールを実行して結果を追加
	for _, tc := range toolCalls {
		agent.executeToolOnly(tc)
	}

	// 合計: assistant(1) + tool(2) = 3 メッセージ
	totalAdded := len(agent.History) - initialLen
	if totalAdded != 3 {
		t.Fatalf("Parallel FC added %d messages, want 3 (1 assistant + 2 tool)", totalAdded)
	}

	// メッセージ構造を検証
	base := initialLen

	// [0] assistant with 2 ToolCalls
	assistantMsg := agent.History[base]
	if assistantMsg.Role != "assistant" {
		t.Errorf("History[%d].Role = %q, want 'assistant'", base, assistantMsg.Role)
	}
	if len(assistantMsg.ToolCalls) != 2 {
		t.Fatalf("History[%d].ToolCalls length = %d, want 2", base, len(assistantMsg.ToolCalls))
	}

	// [1] tool result for call_1
	tool1 := agent.History[base+1]
	if tool1.Role != "tool" {
		t.Errorf("History[%d].Role = %q, want 'tool'", base+1, tool1.Role)
	}
	if tool1.ToolCallID != "call_1" {
		t.Errorf("History[%d].ToolCallID = %q, want 'call_1'", base+1, tool1.ToolCallID)
	}

	// [2] tool result for call_2
	tool2 := agent.History[base+2]
	if tool2.Role != "tool" {
		t.Errorf("History[%d].Role = %q, want 'tool'", base+2, tool2.Role)
	}
	if tool2.ToolCallID != "call_2" {
		t.Errorf("History[%d].ToolCallID = %q, want 'call_2'", base+2, tool2.ToolCallID)
	}

	// AssistantMessages は1回のみ
	if agent.Stats.AssistantMessages != 1 {
		t.Errorf("AssistantMessages = %d, want 1", agent.Stats.AssistantMessages)
	}
}

// TestAdjustSplitForFCPairs_ParallelFC はパラレル FC 構造（assistant → tool × N）で
// 分割点が正しく調整されることを確認
func TestAdjustSplitForFCPairs_ParallelFC(t *testing.T) {
	// パラレル FC: 1つの assistant に2つの ToolCall、2つの tool result
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"paths":["/a.go"]}`}},
			{ID: "call_2", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"paths":["/b.go"]}`}},
		}},
		{Role: "tool", Content: "file a content", ToolCallID: "call_1"},
		{Role: "tool", Content: "file b content", ToolCallID: "call_2"},
		{Role: "user", Content: "msg2"},
	}

	// splitIdx=3 → toKeep[0] = role:"tool" → tool を巻き戻し → assistant まで戻る
	got := adjustSplitForFCPairs(history, 3)
	if got != 1 {
		t.Errorf("adjustSplitForFCPairs(parallel FC, splitIdx=3) = %d, want 1", got)
	}

	// splitIdx=2 → toKeep[0] = role:"tool" → 巻き戻し
	got = adjustSplitForFCPairs(history, 2)
	if got != 1 {
		t.Errorf("adjustSplitForFCPairs(parallel FC, splitIdx=2) = %d, want 1", got)
	}
}
