package agent

import "github.com/susugadx/xelyon-cli/internal/api"

const providerHistoryReductionLatestToolOutput = "latest read_file output"

func providerHistoryReductionRequestHistory(callID, oldRead string) []api.Message {
	return []api.Message{
		providerHistoryAssistantToolCall(callID, "read_file"),
		providerHistoryToolResult(callID, "read_file", oldRead),
		{Role: "assistant", Content: "after old read"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", providerHistoryReductionLatestToolOutput),
		{Role: "assistant", Content: "done"},
	}
}
