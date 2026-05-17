package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestAdjustSplitForFCPairs_ToolAtKeepHead(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"/a.go"}`}},
		}},
		{Role: "tool", Content: "file content", ToolCallID: "call_1"},
		{Role: "user", Content: "msg3"},
	}

	got := adjustSplitForFCPairs(history, 3)
	if got != 1 {
		t.Errorf("Expected splitIdx=1, got %d", got)
	}
}

func TestAdjustSplitForFCPairs_AssistantAtCompressTail(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "bash", Arguments: `{"command":"ls"}`}},
		}},
		{Role: "tool", Content: "output", ToolCallID: "call_1"},
		{Role: "user", Content: "msg2"},
	}

	got := adjustSplitForFCPairs(history, 2)
	if got != 1 {
		t.Errorf("Expected splitIdx=1, got %d", got)
	}
}

func TestAdjustSplitForFCPairs_ZeroBoundary(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
	}

	got := adjustSplitForFCPairs(history, 0)
	if got != 0 {
		t.Errorf("Expected splitIdx=0 (boundary guard), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_MinBoundaryWithTool(t *testing.T) {
	history := []api.Message{
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
		{Role: "user", Content: "msg"},
	}

	got := adjustSplitForFCPairs(history, 1)
	if got != 2 {
		t.Errorf("Expected splitIdx=2 (FC pair preserved in toCompress), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_EntireHistoryIsOneFC(t *testing.T) {
	history := []api.Message{
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "bash", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
	}

	got := adjustSplitForFCPairs(history, 1)
	if got != 0 {
		t.Errorf("Expected splitIdx=0 (unsplittable), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_ParallelToolsAtHead(t *testing.T) {
	history := []api.Message{
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "bash", Arguments: `{}`}},
			{ID: "call_2", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "result1", ToolCallID: "call_1"},
		{Role: "tool", Content: "result2", ToolCallID: "call_2"},
		{Role: "user", Content: "next"},
	}

	got := adjustSplitForFCPairs(history, 1)
	if got != 3 {
		t.Errorf("Expected splitIdx=3 (parallel FC pair preserved), got %d", got)
	}
}

func TestAdjustSplitForFCPairs_NoFC(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "user", Content: "msg3"},
		{Role: "assistant", Content: "msg4"},
		{Role: "user", Content: "msg5"},
	}

	got := adjustSplitForFCPairs(history, 3)
	if got != 3 {
		t.Errorf("Expected splitIdx=3 (unchanged), got %d", got)
	}
}

func TestSplitHistoryForCompression_UsesPersistableHistoryForCompressedSide(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "runtime old"},
		{Role: "assistant", Content: "runtime middle"},
		{Role: "user", Content: "runtime latest"},
	}
	persistHistory := []api.Message{
		{Role: "user", Content: "persist old"},
		{Role: "assistant", Content: "persist middle"},
		{Role: "user", Content: "persist latest"},
	}

	split := splitHistoryForCompression(history, persistHistory, 1)

	if len(split.toCompress) != 2 {
		t.Fatalf("toCompress len = %d, want 2", len(split.toCompress))
	}
	if split.toCompress[0].Content != "persist old" || split.toCompress[1].Content != "persist middle" {
		t.Fatalf("toCompress = %#v, want persistable compressed messages", split.toCompress)
	}
	if len(split.toKeep) != 1 || split.toKeep[0].Content != "runtime latest" {
		t.Fatalf("toKeep = %#v, want runtime tail", split.toKeep)
	}
	if len(split.toKeepPersist) != 1 || split.toKeepPersist[0].Content != "persist latest" {
		t.Fatalf("toKeepPersist = %#v, want persistable tail", split.toKeepPersist)
	}
}

func TestSplitHistoryForCompression_FCPairProtectionCanMakeCompressionEmpty(t *testing.T) {
	history := []api.Message{
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "result", ToolCallID: "call_1"},
	}

	split := splitHistoryForCompression(history, history, 1)

	if len(split.toCompress) != 0 {
		t.Fatalf("toCompress len = %d, want 0 for unsplittable FC pair", len(split.toCompress))
	}
	if len(split.toKeep) != 2 {
		t.Fatalf("toKeep len = %d, want 2", len(split.toKeep))
	}
}

func TestSplitHistoryBeforeIndex_KeepsCurrentTurnTail(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "old user"},
		{Role: "assistant", Content: "old assistant"},
		{Role: "user", Content: "current user"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "current tool result", ToolCallID: "call_1"},
	}
	persistHistory := []api.Message{
		{Role: "user", Content: "old user persisted"},
		{Role: "assistant", Content: "old assistant persisted"},
		{Role: "user", Content: "current user persisted"},
		{Role: "assistant", Content: "", ToolCalls: history[3].ToolCalls},
		{Role: "tool", Content: "current tool result persisted", ToolCallID: "call_1"},
	}

	split := splitHistoryBeforeIndex(history, persistHistory, 2)

	if len(split.toCompress) != 2 {
		t.Fatalf("toCompress len = %d, want 2", len(split.toCompress))
	}
	if split.toCompress[0].Content != "old user persisted" || split.toCompress[1].Content != "old assistant persisted" {
		t.Fatalf("toCompress = %#v, want persisted pre-turn history", split.toCompress)
	}
	if len(split.toKeep) != 3 || split.toKeep[0].Content != "current user" || split.toKeep[2].Content != "current tool result" {
		t.Fatalf("toKeep = %#v, want current runtime turn tail", split.toKeep)
	}
	if len(split.toKeepPersist) != 3 || split.toKeepPersist[0].Content != "current user persisted" || split.toKeepPersist[2].Content != "current tool result persisted" {
		t.Fatalf("toKeepPersist = %#v, want current persisted turn tail", split.toKeepPersist)
	}
}

func TestSplitHistoryForInTurnCompression_RespectsKeepRecentBeforeCurrentTurn(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "old 0"},
		{Role: "assistant", Content: "old 1"},
		{Role: "user", Content: "old 2"},
		{Role: "assistant", Content: "old 3"},
		{Role: "user", Content: "recent old 4"},
		{Role: "assistant", Content: "recent old 5"},
		{Role: "user", Content: "current user"},
		{Role: "assistant", Content: "current assistant"},
	}
	persistHistory := []api.Message{
		{Role: "user", Content: "persist old 0"},
		{Role: "assistant", Content: "persist old 1"},
		{Role: "user", Content: "persist old 2"},
		{Role: "assistant", Content: "persist old 3"},
		{Role: "user", Content: "persist recent old 4"},
		{Role: "assistant", Content: "persist recent old 5"},
		{Role: "user", Content: "persist current user"},
		{Role: "assistant", Content: "persist current assistant"},
	}

	split := splitHistoryForInTurnCompression(history, persistHistory, 6, 4)

	if len(split.toCompress) != 4 || split.toCompress[0].Content != "persist old 0" || split.toCompress[3].Content != "persist old 3" {
		t.Fatalf("toCompress = %#v, want only messages older than keep_recent", split.toCompress)
	}
	if len(split.toKeep) != 4 || split.toKeep[0].Content != "recent old 4" || split.toKeep[2].Content != "current user" {
		t.Fatalf("toKeep = %#v, want keep_recent tail plus current turn", split.toKeep)
	}
	if len(split.toKeepPersist) != 4 || split.toKeepPersist[0].Content != "persist recent old 4" || split.toKeepPersist[2].Content != "persist current user" {
		t.Fatalf("toKeepPersist = %#v, want persisted keep_recent tail plus current turn", split.toKeepPersist)
	}
}

func TestSplitHistoryForInTurnCompression_NoPreTurnHistoryKeepsCurrentTurn(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "current user"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{}`}},
		}},
		{Role: "tool", Content: "current tool result", ToolCallID: "call_1"},
	}
	persistHistory := []api.Message{
		{Role: "user", Content: "persist current user"},
		{Role: "assistant", Content: "", ToolCalls: history[1].ToolCalls},
		{Role: "tool", Content: "persist current tool result", ToolCallID: "call_1"},
	}

	split := splitHistoryForInTurnCompression(history, persistHistory, 0, 1)

	if len(split.toCompress) != 0 {
		t.Fatalf("toCompress = %#v, want no compression when there is no pre-turn history", split.toCompress)
	}
	if len(split.toKeep) != 3 || split.toKeep[0].Content != "current user" || split.toKeep[2].Content != "current tool result" {
		t.Fatalf("toKeep = %#v, want full current runtime turn", split.toKeep)
	}
	if len(split.toKeepPersist) != 3 || split.toKeepPersist[0].Content != "persist current user" || split.toKeepPersist[2].Content != "persist current tool result" {
		t.Fatalf("toKeepPersist = %#v, want full current persisted turn", split.toKeepPersist)
	}
}

func TestSplitHistoryForInTurnCompression_KeepsCurrentTurnWhenLongerThanKeepRecent(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "old 0"},
		{Role: "assistant", Content: "old 1"},
		{Role: "user", Content: "current 0"},
		{Role: "assistant", Content: "current 1"},
		{Role: "tool", Content: "current 2"},
	}

	split := splitHistoryForInTurnCompression(history, history, 2, 1)

	if len(split.toCompress) != 2 {
		t.Fatalf("toCompress len = %d, want 2 pre-turn messages", len(split.toCompress))
	}
	if len(split.toKeep) != 3 || split.toKeep[0].Content != "current 0" {
		t.Fatalf("toKeep = %#v, want full current turn even when longer than keep_recent", split.toKeep)
	}
}

func TestSplitHistoryBeforeIndex_FCPairProtectionKeepsPreviousOpenPairOutOfCompression(t *testing.T) {
	history := []api.Message{
		{Role: "user", Content: "old user"},
		{Role: "assistant", Content: "", ToolCalls: []api.OpenAIToolCall{
			{ID: "call_1", Function: api.OpenAIToolCallFunction{Name: "bash", Arguments: `{}`}},
		}},
		{Role: "user", Content: "current user"},
		{Role: "assistant", Content: "current assistant"},
	}

	split := splitHistoryBeforeIndex(history, history, 2)

	if len(split.toCompress) != 1 || split.toCompress[0].Content != "old user" {
		t.Fatalf("toCompress = %#v, want only messages before previous open FC pair", split.toCompress)
	}
	if len(split.toKeep) != 3 || len(split.toKeep[0].ToolCalls) != 1 || split.toKeep[1].Content != "current user" {
		t.Fatalf("toKeep = %#v, want previous open FC pair plus current tail", split.toKeep)
	}
}
