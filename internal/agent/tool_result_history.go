package agent

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) appendToolResultToHistory(toolCall *tools.ToolCall, result string) {
	if toolCall == nil {
		return
	}
	a.appendToolResultToHistoryWithContent(toolCall, result, toolruntime.FormatTextToolResultContent(toolCall.Tool, result))
}

func (a *Agent) appendToolResultToHistoryWithContent(toolCall *tools.ToolCall, functionContent, textContent string) {
	if a == nil || toolCall == nil {
		return
	}

	decision := toolResultRetentionDecisionFor(toolCall)
	msg := toolruntime.BuildToolResultMessage(toolCall, functionContent, textContent)
	if toolCall.ID != "" {
		msg.SetOpenAIResponsesInputItems([]api.InputItem{{
			Type:   "function_call_output",
			CallID: toolCall.ID,
			Output: functionContent,
		}})
	}
	if decision.KeepHistory {
		a.History = append(a.History, msg)
	}

	if !decision.KeepSessionConversation {
		return
	}
	if toolCall.ID != "" {
		a.appendSessionMessageFromAPI(msg, a.CurrentModel)
		return
	}
	a.appendSessionMessage(msg.Role, msg.Content, a.CurrentModel)
}
