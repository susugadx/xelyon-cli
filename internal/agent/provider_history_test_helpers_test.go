package agent

import (
	"encoding/json"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func providerHistoryToolCall(id, name string) api.OpenAIToolCall {
	return providerHistoryToolCallWithArguments(id, name, "{}")
}

func providerHistoryToolCallWithArguments(id, name, arguments string) api.OpenAIToolCall {
	return api.OpenAIToolCall{
		ID:       id,
		Type:     "function",
		Function: api.OpenAIToolCallFunction{Name: name, Arguments: arguments},
	}
}

func providerHistoryToolCallWithJSONArguments(t *testing.T, id, name string, arguments map[string]string) api.OpenAIToolCall {
	t.Helper()
	return providerHistoryToolCallWithArguments(id, name, providerHistoryJSONArguments(t, arguments))
}

func providerHistoryJSONArguments(t *testing.T, arguments map[string]string) string {
	t.Helper()
	data, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("json.Marshal(%#v) error = %v", arguments, err)
	}
	return string(data)
}

func providerHistoryAssistantToolCall(id, name string) api.Message {
	return providerHistoryAssistantToolCalls(providerHistoryToolCall(id, name))
}

func providerHistoryAssistantToolCalls(toolCalls ...api.OpenAIToolCall) api.Message {
	return api.Message{
		Role:      "assistant",
		Content:   "calling tools",
		ToolCalls: toolCalls,
	}
}

func providerHistoryToolResult(id, name, content string) api.Message {
	return api.Message{
		Role:       "tool",
		Content:    content,
		ToolCallID: id,
		ToolName:   name,
	}
}
