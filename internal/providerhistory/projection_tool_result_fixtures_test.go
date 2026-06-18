package providerhistory

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func providerHistoryTestWebSearchHistory(t *testing.T, callID, query, content string, duplicate bool) []api.Message {
	t.Helper()
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, callID, "web_search", map[string]string{"query": query})),
		providerHistoryTestToolResult(callID, "web_search", content),
		{Role: "assistant", Content: "after web search"},
	}
	if duplicate {
		history = append(history,
			providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_web_later", "web_search", map[string]string{"query": query})),
			providerHistoryTestToolResult("call_web_later", "web_search", content),
			api.Message{Role: "assistant", Content: "after duplicate"},
		)
	}
	history = append(history,
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		api.Message{Role: "assistant", Content: "done"},
	)
	return history
}

func providerHistoryTestActivateSkillHistory(t *testing.T, oldBody, newBody string) []api.Message {
	t.Helper()
	history := []api.Message{
		providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill", "activate_skill", map[string]string{"name": "goal-plan-author"})),
		providerHistoryTestToolResult("call_skill", "activate_skill", oldBody),
		{Role: "assistant", Content: "after skill"},
	}
	if newBody != "" {
		history = append(history,
			providerHistoryTestAssistantToolCalls(providerHistoryTestToolCallWithJSONArguments(t, "call_skill_later", "activate_skill", map[string]string{"name": "goal-plan-author"})),
			providerHistoryTestToolResult("call_skill_later", "activate_skill", newBody),
			api.Message{Role: "assistant", Content: "after later skill"},
		)
	}
	history = append(history,
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		api.Message{Role: "assistant", Content: "done"},
	)
	return history
}

func providerHistoryTestLargeWebSearchResult() string {
	return strings.Repeat(`1. OpenAI Responses API guide
URL: https://platform.openai.com/docs/guides/responses
The official docs describe response ids and follow-up calls.
2. OpenAI API reference
URL: https://platform.openai.com/docs/api-reference/responses
Reference material for Responses API fields.
`, 120)
}

func providerHistoryTestLargeSkillBody(name string) string {
	return "# " + name + "\n\n" + strings.Repeat("Skill instruction line for "+name+".\n", 260)
}

func providerHistoryTestMCPHistory(callID, content string) []api.Message {
	return []api.Message{
		providerHistoryTestAssistantToolCall(callID, "mcp_context7_get_library_docs"),
		providerHistoryTestToolResult(callID, "mcp_context7_get_library_docs", content),
		{Role: "assistant", Content: "after mcp result"},
		providerHistoryTestAssistantToolCall("call_latest", "read_file"),
		providerHistoryTestToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}
}

func providerHistoryTestLargeSafeMCPResult() string {
	return `{"items":[` + strings.Repeat(`{"title":"public metadata","value":"safe documentation result","score":1},`, 2600) + `{"title":"tail","value":"safe"}]}`
}

func providerHistoryTestLargeSensitiveMCPResult() string {
	return `{"items":[` + strings.Repeat(`{"title":"private issue body","email":"customer@example.test","token":"secret-token","value":"customer private message body"},`, 2600) + `{"title":"tail","value":"private customer"}]}`
}
