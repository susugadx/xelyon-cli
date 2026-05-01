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
