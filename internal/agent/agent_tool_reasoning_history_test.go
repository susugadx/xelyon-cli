package agent

import (
	"encoding/json"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/history"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type reasoningMockProvider struct {
	mockProvider
	reasoning string
}

func (m *reasoningMockProvider) LastReasoningContent() string {
	return m.reasoning
}

func TestAgentToolLoopHistoryPersistsReasoningForOpenAICompatReplay(t *testing.T) {
	provider := &reasoningMockProvider{
		mockProvider: mockProvider{name: "test"},
		reasoning:    "Need to inspect README before answering.",
	}
	agent := &Agent{
		CurrentModel:    "test-model",
		CurrentProvider: provider,
		agentConversationState: agentConversationState{
			session: history.NewSession("test-model"),
		},
	}
	toolCall := &tools.ToolCall{
		ID:      "call_1",
		Tool:    "read_file",
		Args:    map[string]string{"path": "README.md"},
		RawArgs: map[string]any{"path": "README.md"},
	}

	agent.addToolCallsToHistory("I'll inspect README.", []*tools.ToolCall{toolCall})
	agent.appendToolResultToHistory(toolCall, "README contents")

	restored := agent.session.ToAPIMessages()
	if len(restored) != 2 {
		t.Fatalf("len(session.ToAPIMessages()) = %d, want 2", len(restored))
	}
	assistant := restored[0]
	if assistant.Role != "assistant" || assistant.ReasoningContent != "Need to inspect README before answering." {
		t.Fatalf("assistant message = %+v, want reasoning_content preserved", assistant)
	}
	if len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].ID != "call_1" || assistant.ToolCalls[0].Function.Name != "read_file" {
		t.Fatalf("assistant ToolCalls = %+v, want read_file call_1 preserved", assistant.ToolCalls)
	}
	tool := restored[1]
	if tool.Role != "tool" || tool.ToolCallID != "call_1" || tool.Content != "README contents" {
		t.Fatalf("tool message = %+v, want role=tool with tool_call_id/content", tool)
	}

	payload := marshalAgentMessagesPayload(t, openaicompat.BuildChatMessages("system", restored))
	if payload[1]["reasoning_content"] != "Need to inspect README before answering." {
		t.Fatalf("payload assistant reasoning_content = %#v, want preserved", payload[1]["reasoning_content"])
	}
	if _, ok := payload[1]["tool_calls"].([]any); !ok {
		t.Fatalf("payload assistant tool_calls = %#v, want tool_calls array", payload[1]["tool_calls"])
	}
	if payload[2]["role"] != "tool" || payload[2]["tool_call_id"] != "call_1" || payload[2]["content"] != "README contents" {
		t.Fatalf("payload tool message = %#v, want role/tool_call_id/content preserved", payload[2])
	}
}

func marshalAgentMessagesPayload(t *testing.T, messages []api.Message) []map[string]any {
	t.Helper()
	payload, err := json.Marshal(messages)
	if err != nil {
		t.Fatalf("json.Marshal(messages) error = %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(messages) error = %v", err)
	}
	return decoded
}
