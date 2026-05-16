package agent

import "github.com/susugadx/xelyon-cli/internal/api"

func providerHistoryToolCall(id, name string) api.OpenAIToolCall {
	return api.OpenAIToolCall{
		ID:       id,
		Type:     "function",
		Function: api.OpenAIToolCallFunction{Name: name, Arguments: "{}"},
	}
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
