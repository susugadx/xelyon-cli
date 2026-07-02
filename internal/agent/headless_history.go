package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func assignHeadlessRescueToolCallIDs(toolCalls []*tools.ToolCall) {
	for i, tc := range toolCalls {
		if tc == nil || tc.ID != "" {
			continue
		}
		toolCalls[i].ID = fmt.Sprintf("call_rescue_%03d", i+1)
	}
}

func appendHeadlessToolCallsToHistory(agent *Agent, response string, toolCalls []*tools.ToolCall) {
	if agent == nil || len(toolCalls) == 0 {
		return
	}

	explanation, _ := extractExplanationAndTool(response)
	reasoningContent := agent.getLastReasoningContent()
	contentBlocks := agent.getLastAnthropicContentBlocks()
	openAIToolCalls := buildOpenAIToolCallsForHistory(toolCalls)
	appendAssistantToolCallsHistoryMessage(agent, explanation, reasoningContent, contentBlocks, openAIToolCalls)
	if agent.Stats != nil {
		agent.Stats.AssistantMessages++
	}
}

func appendHeadlessToolResultToHistory(agent *Agent, toolCall *tools.ToolCall, result string) {
	if agent == nil || toolCall == nil {
		return
	}
	if !keepToolResultHistory(toolCall) {
		return
	}
	agent.History = append(agent.History, toolruntime.BuildToolResultMessage(toolCall, result, toolruntime.FormatTextToolResultContent(toolCall.Tool, result)))
}
