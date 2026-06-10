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

type responsesReplayMockProvider struct {
	mockProvider
	items []api.InputItem
}

func (m *responsesReplayMockProvider) LastOpenAIResponsesInputItems() []api.InputItem {
	return api.CloneInputItems(m.items)
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

func TestAgentToolLoopHistoryKeepsReasoningToolCallsAndToolResultInMemory(t *testing.T) {
	provider := &reasoningMockProvider{
		mockProvider: mockProvider{name: "test"},
		reasoning:    "調査します",
	}
	agent := &Agent{
		CurrentProvider: provider,
	}
	toolCall := &tools.ToolCall{
		ID:      "call_1",
		Tool:    "read_file",
		Args:    map[string]string{"path": "README.md"},
		RawArgs: map[string]any{"path": "README.md"},
	}

	agent.addToolCallsToHistory("README を確認します。", []*tools.ToolCall{toolCall})
	agent.appendToolResultToHistory(toolCall, "README contents")

	if len(agent.History) != 2 {
		t.Fatalf("len(agent.History) = %d, want assistant tool_calls + tool result", len(agent.History))
	}
	assistant := agent.History[0]
	if assistant.Role != "assistant" || assistant.ReasoningContent != "調査します" {
		t.Fatalf("assistant history = %+v, want assistant with reasoning_content", assistant)
	}
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("assistant ToolCalls len = %d, want 1", len(assistant.ToolCalls))
	}
	if assistant.ToolCalls[0].ID != "call_1" ||
		assistant.ToolCalls[0].Type != "function" ||
		assistant.ToolCalls[0].Function.Name != "read_file" ||
		assistant.ToolCalls[0].Function.Arguments != `{"path":"README.md"}` {
		t.Fatalf("assistant ToolCalls[0] = %+v, want read_file call_1 with arguments", assistant.ToolCalls[0])
	}
	toolResult := agent.History[1]
	if toolResult.Role != "tool" || toolResult.ToolCallID != "call_1" || toolResult.ToolName != "read_file" || toolResult.Content != "README contents" {
		t.Fatalf("tool result history = %+v, want role=tool with tool_call_id/name/content", toolResult)
	}
}

func TestAgentToolLoopHistoryKeepsOpenAIResponsesReplayItems(t *testing.T) {
	provider := &responsesReplayMockProvider{
		mockProvider: mockProvider{name: "test"},
		items: []api.InputItem{
			{Type: "message", Role: "assistant", ID: "msg_1", Content: "I'll inspect README."},
			{Type: "function_call", ID: "fc_1", CallID: "call_1", Name: "read_file", Arguments: `{"path":"README.md"}`},
		},
	}
	agent := &Agent{
		CurrentProvider: provider,
		CurrentModel:    "test-model",
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

	if len(agent.History) != 2 {
		t.Fatalf("len(agent.History) = %d, want assistant + tool result", len(agent.History))
	}
	assistantReplay := agent.History[0].OpenAIResponsesInputItems()
	if len(assistantReplay) != 2 || assistantReplay[0].ID != "msg_1" || assistantReplay[1].CallID != "call_1" {
		t.Fatalf("assistant replay items = %#v, want provider replay message + function_call", assistantReplay)
	}
	toolReplay := agent.History[1].OpenAIResponsesInputItems()
	if len(toolReplay) != 1 || toolReplay[0].Type != "function_call_output" || toolReplay[0].CallID != "call_1" || toolReplay[0].Output != "README contents" {
		t.Fatalf("tool replay items = %#v, want function_call_output", toolReplay)
	}

	restored := agent.session.ToAPIMessages()
	items := api.ConvertHistoryToInputItems(restored)
	if len(items) != 3 {
		t.Fatalf("len(ConvertHistoryToInputItems(restored)) = %d, want replay message/function_call/function_call_output", len(items))
	}
	if items[0].ID != "msg_1" || items[1].CallID != "call_1" || items[2].Type != "function_call_output" {
		t.Fatalf("converted replay items = %#v", items)
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
