package providerhistory

import (
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerhistory/editargs"
)

func applyProviderHistoryEditArgReplacementCandidate(report *CommandEditDryRunReport, candidateIndex int, toolName string, ref providerHistoryAssistantToolCallRef, toolResultContent string, projection []api.Message) {
	if report == nil || candidateIndex < 0 || candidateIndex >= len(report.Candidates) {
		return
	}
	candidate := report.Candidates[candidateIndex]
	if candidate.Kind != "edit_arguments" {
		return
	}
	if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
		return
	}
	if !providerHistoryCommandProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
		return
	}
	if ref.historyIndex < 0 || ref.historyIndex >= len(projection) {
		return
	}

	payload, keepReason := editargs.Payload(toolName, ref.arguments)
	if keepReason != "" || payload.Reason != candidate.Reason {
		return
	}
	replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          toolName,
		Arguments:         ref.arguments,
		ToolResultContent: toolResultContent,
	})
	if !ok {
		return
	}
	if !applyProviderHistoryEditArgReplacementProjection(&projection[ref.historyIndex], candidate.ToolCallID, replacement) {
		return
	}

	report.EditArgReplacedCount++
	report.EditArgReplacementSavedBytes += replacement.SavedBytes
	report.ApproxEditArgReplacementSavedTokens += replacement.SavedTokens
	report.ReplacementStatus = providerHistoryCommandEditReplacementStatusPartialApply
}

func applyProviderHistoryEditArgReplacementProjection(msg *api.Message, toolCallID string, replacement editargs.Replacement) bool {
	if msg == nil || msg.Role != "assistant" || toolCallID == "" || replacement.ToolName == "" {
		return false
	}
	toolCallIndex := providerHistoryEditArgToolCallIndex(*msg, toolCallID, replacement.ToolName)
	if toolCallIndex < 0 {
		return false
	}
	if !syncProviderHistoryEditArgAnthropicState(msg, toolCallID, replacement) {
		return false
	}
	if !msg.ReplaceOpenAIResponsesFunctionCallArguments(toolCallID, replacement.ToolName, replacement.Arguments) {
		return false
	}

	msg.ToolCalls[toolCallIndex].Function.Arguments = replacement.Arguments
	return true
}

func providerHistoryEditArgToolCallIndex(msg api.Message, toolCallID, toolName string) int {
	for i, toolCall := range msg.ToolCalls {
		if toolCall.ID != toolCallID {
			continue
		}
		if toolCall.Function.Name != toolName {
			return -1
		}
		return i
	}
	return -1
}

func syncProviderHistoryEditArgAnthropicState(msg *api.Message, toolCallID string, replacement editargs.Replacement) bool {
	if msg == nil {
		return false
	}
	blocks := msg.AnthropicContentBlocks()
	if len(blocks) == 0 {
		return true
	}

	updated := false
	for i := range blocks {
		block := &blocks[i]
		if block.Type != "tool_use" || block.ID != toolCallID {
			continue
		}
		if updated && replacement.ToolName != "write_file" {
			return false
		}
		if !providerHistoryEditArgAnthropicToolNameMatches(block.Name, replacement.ToolName) {
			return false
		}
		if len(block.Input) == 0 || !replacement.ApplyAnthropicInput(block.Input) {
			return false
		}
		updated = true
	}
	if !updated {
		return false
	}
	msg.SetAnthropicContentBlocks(blocks)
	return true
}

func providerHistoryEditArgAnthropicToolNameMatches(blockName, replacementToolName string) bool {
	if blockName == replacementToolName {
		return true
	}
	return blockName == "" && replacementToolName == "write_file"
}
