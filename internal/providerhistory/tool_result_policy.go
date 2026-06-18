package providerhistory

import (
	"fmt"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/providerhistory/toolresults"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func isProviderHistoryReductionAlwaysKeptTool(toolName string) bool {
	if providerHistoryIsCommandEditTool(toolName) {
		return true
	}
	return tools.IsWriteTool(toolName)
}

func isEvidenceBackedReductionTool(toolName string) bool {
	switch toolName {
	case "read_file", "search_code", "gather_context":
		return true
	default:
		return false
	}
}

func isStructuredToolResultReductionTool(toolName string) bool {
	switch toolName {
	case "list_dir", "activate_skill":
		return true
	default:
		return false
	}
}

func isToolResultReductionCandidateTool(toolName string) bool {
	return isEvidenceBackedReductionTool(toolName) || isStructuredToolResultReductionTool(toolName)
}

func providerHistoryReductionCandidateReason(toolName string) string {
	if toolName == "activate_skill" {
		return "old_duplicate_activate_skill_result"
	}
	if toolName == "web_search" {
		return "old_duplicate_web_search_result"
	}
	if isStructuredToolResultReductionTool(toolName) {
		return "old_successful_" + toolName + "_result"
	}
	return "old_tool_result_after_assistant_turn"
}

func providerHistoryReductionSuggestedReplacementText(candidate ReductionCandidate, arguments, content string, messages []api.Message) string {
	if isStructuredToolResultReductionTool(candidate.ToolName) {
		replacement, _, ok := toolresults.BuildStructuredReplacement(toolresults.NewReplacementRequestWithMessages(candidate.ToolName, arguments, content, candidate.ToolCallID, candidate.HistoryIndex, messages))
		if !ok {
			return ""
		}
		return replacement.Text()
	}
	return fmt.Sprintf("[omitted old %s result; see evidence pointer]", candidate.ToolName)
}

func providerHistoryReductionEntry(historyIndex int, msg api.Message) ReductionCandidate {
	return ReductionCandidate{
		HistoryIndex:     historyIndex,
		Role:             msg.Role,
		ToolName:         msg.ToolName,
		ToolCallID:       msg.ToolCallID,
		OriginalByteSize: len(msg.Content),
		OriginalRuneSize: utf8.RuneCountInString(msg.Content),
	}
}
