package agent

import (
	"fmt"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func buildProviderHistoryReductionDetectionReport(original, projected []api.Message, mode ProviderHistoryReductionMode) ProviderHistoryProjectionReport {
	report := ProviderHistoryProjectionReport{
		Mode:                  mode,
		OriginalMessageCount:  len(original),
		ProjectedMessageCount: len(projected),
	}
	if len(original) == 0 {
		return report
	}

	assistantToolCallsByID := collectProviderHistoryAssistantToolCalls(original)
	trailingToolStart := providerHistoryTrailingToolSuffixStart(original)
	latestToolResultIndex := providerHistoryLatestToolResultIndex(original)

	for i, msg := range original {
		if msg.Role != "tool" {
			continue
		}
		report.ToolResultCount++
		entry := providerHistoryReductionEntry(i, msg)

		if i >= trailingToolStart {
			entry.KeepReason = "trailing_tool_suffix"
			report.Kept = append(report.Kept, entry)
			continue
		}
		if i == latestToolResultIndex {
			entry.KeepReason = "latest_tool_result"
			report.Kept = append(report.Kept, entry)
			continue
		}
		if msg.ToolCallID == "" {
			entry.KeepReason = "missing_tool_call_id"
			report.Kept = append(report.Kept, entry)
			continue
		}
		localToolResultCount := countProviderHistoryToolResultsByIDInContiguousBlock(original, i, msg.ToolCallID)
		if localToolResultCount > 1 {
			entry.KeepReason = "ambiguous_tool_result_id"
			report.Kept = append(report.Kept, entry)
			continue
		}

		allRefs := assistantToolCallsByID[msg.ToolCallID]
		localAssistantIndex := providerHistoryContiguousAssistantIndexForToolResult(original, i)
		localRefs := providerHistoryAssistantToolCallRefsAtIndex(original, localAssistantIndex, msg.ToolCallID)
		if len(localRefs) == 0 {
			if providerHistoryHasEarlierAssistantToolCallRef(allRefs, i) {
				entry.KeepReason = "non_contiguous_tool_call_linkage"
				report.Kept = append(report.Kept, entry)
				continue
			}
			entry.KeepReason = "missing_assistant_tool_call"
			report.Kept = append(report.Kept, entry)
			continue
		}
		if len(localRefs) > 1 {
			entry.KeepReason = "ambiguous_assistant_tool_call"
			report.Kept = append(report.Kept, entry)
			continue
		}
		ref := localRefs[0]

		toolName := msg.ToolName
		if toolName == "" {
			toolName = ref.name
		} else if toolName != ref.name {
			entry.ToolName = toolName
			entry.KeepReason = "mismatched_tool_name"
			report.Kept = append(report.Kept, entry)
			continue
		}
		entry.ToolName = toolName
		if toolName == "" {
			entry.KeepReason = "missing_tool_name"
			report.Kept = append(report.Kept, entry)
			continue
		}

		if isProviderHistoryReductionAlwaysKeptTool(toolName) {
			entry.KeepReason = "write_or_command_tool"
			report.Kept = append(report.Kept, entry)
			continue
		}
		if !isProviderHistoryReductionCandidateTool(toolName) {
			entry.KeepReason = "tool_not_in_reduction_allowlist"
			report.Kept = append(report.Kept, entry)
			continue
		}
		if !providerHistoryHasLaterAssistant(original, i) {
			entry.KeepReason = "no_later_assistant_message"
			report.Kept = append(report.Kept, entry)
			continue
		}

		entry.Reason = "old_tool_result_after_assistant_turn"
		entry.SuggestedReplacementKind = providerHistoryReductionReplacementKind(toolName)
		entry.SuggestedReplacementText = fmt.Sprintf("[omitted old %s result; see evidence pointer]", toolName)
		report.Candidates = append(report.Candidates, entry)
		if mode == ProviderHistoryReductionDryRun {
			kept := entry
			kept.KeepReason = "dry_run"
			report.Kept = append(report.Kept, kept)
		}
	}

	report.CandidateCount = len(report.Candidates)
	return report
}

type providerHistoryAssistantToolCallRef struct {
	historyIndex int
	name         string
}

func collectProviderHistoryAssistantToolCalls(messages []api.Message) map[string][]providerHistoryAssistantToolCallRef {
	refsByID := make(map[string][]providerHistoryAssistantToolCallRef)
	for i, msg := range messages {
		if msg.Role != "assistant" {
			continue
		}
		for _, toolCall := range msg.ToolCalls {
			if toolCall.ID == "" {
				continue
			}
			refsByID[toolCall.ID] = append(refsByID[toolCall.ID], providerHistoryAssistantToolCallRef{
				historyIndex: i,
				name:         toolCall.Function.Name,
			})
		}
	}
	return refsByID
}

func providerHistoryTrailingToolSuffixStart(messages []api.Message) int {
	start := len(messages)
	for start > 0 && messages[start-1].Role == "tool" {
		start--
	}
	return start
}

func providerHistoryAssistantToolCallRefsAtIndex(messages []api.Message, historyIndex int, toolCallID string) []providerHistoryAssistantToolCallRef {
	if historyIndex < 0 || historyIndex >= len(messages) || messages[historyIndex].Role != "assistant" {
		return nil
	}

	var refs []providerHistoryAssistantToolCallRef
	for _, toolCall := range messages[historyIndex].ToolCalls {
		if toolCall.ID != toolCallID {
			continue
		}
		refs = append(refs, providerHistoryAssistantToolCallRef{
			historyIndex: historyIndex,
			name:         toolCall.Function.Name,
		})
	}
	return refs
}

func providerHistoryHasEarlierAssistantToolCallRef(refs []providerHistoryAssistantToolCallRef, historyIndex int) bool {
	for _, ref := range refs {
		if ref.historyIndex < historyIndex {
			return true
		}
	}
	return false
}

func countProviderHistoryToolResultsByIDInContiguousBlock(messages []api.Message, historyIndex int, toolCallID string) int {
	start := historyIndex
	for start > 0 && messages[start-1].Role == "tool" {
		start--
	}

	count := 0
	for i := start; i < len(messages) && messages[i].Role == "tool"; i++ {
		if messages[i].ToolCallID == toolCallID {
			count++
		}
	}
	return count
}

func providerHistoryContiguousAssistantIndexForToolResult(messages []api.Message, historyIndex int) int {
	for i := historyIndex - 1; i >= 0; i-- {
		if messages[i].Role == "tool" {
			continue
		}
		if messages[i].Role == "assistant" {
			return i
		}
		return -1
	}
	return -1
}

func providerHistoryLatestToolResultIndex(messages []api.Message) int {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "tool" {
			return i
		}
	}
	return -1
}

func providerHistoryHasLaterAssistant(messages []api.Message, historyIndex int) bool {
	for i := historyIndex + 1; i < len(messages); i++ {
		if messages[i].Role == "assistant" {
			return true
		}
	}
	return false
}

func isProviderHistoryReductionAlwaysKeptTool(toolName string) bool {
	switch toolName {
	case "apply_patch", "str_replace", "write_file", "delete_file", "bash", "command":
		return true
	default:
		return tools.IsWriteTool(toolName)
	}
}

func isProviderHistoryReductionCandidateTool(toolName string) bool {
	switch toolName {
	case "read_file", "search_code", "gather_context":
		return true
	default:
		return false
	}
}

func providerHistoryReductionEntry(historyIndex int, msg api.Message) ProviderHistoryReductionCandidate {
	return ProviderHistoryReductionCandidate{
		HistoryIndex:     historyIndex,
		Role:             msg.Role,
		ToolName:         msg.ToolName,
		ToolCallID:       msg.ToolCallID,
		OriginalByteSize: len(msg.Content),
		OriginalRuneSize: utf8.RuneCountInString(msg.Content),
	}
}
