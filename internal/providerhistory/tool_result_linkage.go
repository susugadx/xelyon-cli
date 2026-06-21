package providerhistory

import "github.com/susugadx/xelyon-cli/internal/api"

type providerHistoryAssistantToolCallRef struct {
	historyIndex int
	name         string
	arguments    string
}

type providerHistoryToolResultLinkage struct {
	ToolName     string
	Ref          providerHistoryAssistantToolCallRef
	RefToolNames []string
	KeepReason   string
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
				arguments:    toolCall.Function.Arguments,
			})
		}
	}
	return refsByID
}

func resolveProviderHistoryToolResultLinkage(messages []api.Message, historyIndex int, msg api.Message, assistantToolCallsByID map[string][]providerHistoryAssistantToolCallRef, trailingToolStart, latestToolResultIndex int) providerHistoryToolResultLinkage {
	linkage, localRefs, hasEarlierRef := providerHistoryToolResultLinkageMetadata(messages, historyIndex, msg, assistantToolCallsByID)
	if historyIndex >= trailingToolStart {
		linkage.KeepReason = "trailing_tool_suffix"
		return linkage
	}
	if historyIndex == latestToolResultIndex {
		if linkage.ToolName == "activate_skill" {
			linkage.KeepReason = "activate_skill_latest_activation_keep"
			return linkage
		}
		linkage.KeepReason = "latest_tool_result"
		return linkage
	}
	if msg.ToolCallID == "" {
		linkage.KeepReason = "missing_tool_call_id"
		return linkage
	}
	localToolResultCount := countProviderHistoryToolResultsByIDInContiguousBlock(messages, historyIndex, msg.ToolCallID)
	if localToolResultCount > 1 {
		linkage.KeepReason = "ambiguous_tool_result_id"
		return linkage
	}

	if len(localRefs) == 0 {
		if hasEarlierRef {
			linkage.KeepReason = "non_contiguous_tool_call_linkage"
			return linkage
		}
		linkage.KeepReason = "missing_assistant_tool_call"
		return linkage
	}
	if len(localRefs) > 1 {
		linkage.KeepReason = "ambiguous_assistant_tool_call"
		return linkage
	}

	ref := linkage.Ref
	if msg.ToolName != "" && ref.name != "" && msg.ToolName != ref.name {
		linkage.KeepReason = "mismatched_tool_name"
		return linkage
	}
	return linkage
}

func providerHistoryToolResultLinkageMetadata(messages []api.Message, historyIndex int, msg api.Message, assistantToolCallsByID map[string][]providerHistoryAssistantToolCallRef) (providerHistoryToolResultLinkage, []providerHistoryAssistantToolCallRef, bool) {
	linkage := providerHistoryToolResultLinkage{ToolName: msg.ToolName}
	if msg.ToolCallID == "" {
		return linkage, nil, false
	}

	allRefs := assistantToolCallsByID[msg.ToolCallID]
	localAssistantIndex := providerHistoryContiguousAssistantIndexForToolResult(messages, historyIndex)
	localRefs := providerHistoryAssistantToolCallRefsAtIndex(messages, localAssistantIndex, msg.ToolCallID)
	switch len(localRefs) {
	case 0:
		if providerHistoryHasEarlierAssistantToolCallRef(allRefs, historyIndex) {
			linkage.RefToolNames = providerHistoryToolNamesFromAssistantRefs(allRefs)
			return linkage, localRefs, true
		}
	case 1:
		ref := localRefs[0]
		linkage.Ref = ref
		linkage.RefToolNames = providerHistoryToolNamesFromAssistantRefs(localRefs)
		if linkage.ToolName == "" {
			linkage.ToolName = ref.name
		}
	default:
		linkage.RefToolNames = providerHistoryToolNamesFromAssistantRefs(localRefs)
	}
	return linkage, localRefs, false
}

func providerHistoryToolNamesFromAssistantRefs(refs []providerHistoryAssistantToolCallRef) []string {
	if len(refs) == 0 {
		return nil
	}
	names := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.name == "" {
			continue
		}
		names = append(names, ref.name)
	}
	return names
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
			arguments:    toolCall.Function.Arguments,
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
