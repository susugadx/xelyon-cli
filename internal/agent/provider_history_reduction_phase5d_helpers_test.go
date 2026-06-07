package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func phase5DOutput(label string) string {
	return strings.Repeat(label+"\n", 240)
}

func phase5DToolResultByID(t *testing.T, history []api.Message, callID string) api.Message {
	t.Helper()
	for _, msg := range history {
		if msg.Role == "tool" && msg.ToolCallID == callID {
			return msg
		}
	}
	t.Fatalf("tool result %q not found in history %#v", callID, history)
	return api.Message{}
}

func phase5DReplaceableHistory(callID, toolName, output string) []api.Message {
	return []api.Message{
		providerHistoryAssistantToolCall(callID, toolName),
		providerHistoryToolResult(callID, toolName, output),
		{Role: "assistant", Content: "after old tool"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest raw"),
		{Role: "assistant", Content: "done"},
	}
}
