package toolresults

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func providerHistoryToolResultHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])[:16]
}

func toolArgumentsForToolResultAt(messages []api.Message, toolResultIndex int) string {
	toolCallID := strings.TrimSpace(messages[toolResultIndex].ToolCallID)
	if toolCallID == "" {
		return ""
	}
	assistantIndex := contiguousAssistantIndexForToolResult(messages, toolResultIndex)
	if assistantIndex < 0 {
		return ""
	}
	for _, toolCall := range messages[assistantIndex].ToolCalls {
		if toolCall.ID == toolCallID {
			return toolCall.Function.Arguments
		}
	}
	return ""
}

func toolNameForToolResultAt(messages []api.Message, toolResultIndex int) string {
	toolCallID := strings.TrimSpace(messages[toolResultIndex].ToolCallID)
	if toolCallID == "" {
		return ""
	}
	assistantIndex := contiguousAssistantIndexForToolResult(messages, toolResultIndex)
	if assistantIndex < 0 {
		return ""
	}
	for _, toolCall := range messages[assistantIndex].ToolCalls {
		if toolCall.ID == toolCallID {
			return strings.TrimSpace(toolCall.Function.Name)
		}
	}
	return ""
}

func contiguousAssistantIndexForToolResult(messages []api.Message, toolResultIndex int) int {
	for i := toolResultIndex - 1; i >= 0; i-- {
		switch messages[i].Role {
		case "tool":
			continue
		case "assistant":
			return i
		default:
			return -1
		}
	}
	return -1
}

func singleLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}
