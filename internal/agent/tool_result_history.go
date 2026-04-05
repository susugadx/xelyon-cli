package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func formatTextToolResultContent(toolName, result string) string {
	return fmt.Sprintf("[Tool Result for %s]\n%s", toolName, result)
}

func buildToolResultMessage(toolCall *tools.ToolCall, functionContent, textContent string) api.Message {
	if toolCall == nil {
		return api.Message{}
	}
	if toolCall.ID != "" {
		return api.Message{
			Role:       "tool",
			Content:    functionContent,
			ToolCallID: toolCall.ID,
			ToolName:   toolCall.Tool,
		}
	}
	return api.Message{
		Role:    "user",
		Content: textContent,
	}
}

func (a *Agent) appendToolResultToHistory(toolCall *tools.ToolCall, result string) {
	if toolCall == nil {
		return
	}
	a.appendToolResultToHistoryWithContent(toolCall, result, formatTextToolResultContent(toolCall.Tool, result))
}

func (a *Agent) appendToolResultToHistoryWithContent(toolCall *tools.ToolCall, functionContent, textContent string) {
	if a == nil || toolCall == nil {
		return
	}

	msg := buildToolResultMessage(toolCall, functionContent, textContent)
	a.History = append(a.History, msg)

	if toolCall.ID != "" {
		a.appendSessionMessageFromAPI(msg, a.CurrentModel)
		return
	}
	a.appendSessionMessage(msg.Role, msg.Content, a.CurrentModel)
}
