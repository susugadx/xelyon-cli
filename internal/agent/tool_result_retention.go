package agent

import "github.com/susugadx/xelyon-cli/internal/tools"

type toolResultRetentionDecision struct {
	KeepHistory              bool
	KeepSessionConversation  bool
	KeepSessionToolExecution bool
}

type toolResultRetentionPolicy struct{}

type toolResultRetentionInput struct {
	ToolCall *tools.ToolCall
}

func (toolResultRetentionPolicy) Decide(input toolResultRetentionInput) toolResultRetentionDecision {
	if input.ToolCall == nil {
		return toolResultRetentionDecision{}
	}
	return toolResultRetentionDecision{
		KeepHistory:              true,
		KeepSessionConversation:  true,
		KeepSessionToolExecution: true,
	}
}

func defaultToolResultRetentionPolicy() toolResultRetentionPolicy {
	return toolResultRetentionPolicy{}
}

func toolResultRetentionDecisionFor(toolCall *tools.ToolCall) toolResultRetentionDecision {
	return defaultToolResultRetentionPolicy().Decide(toolResultRetentionInput{ToolCall: toolCall})
}

func keepToolResultHistory(toolCall *tools.ToolCall) bool {
	return toolResultRetentionDecisionFor(toolCall).KeepHistory
}
