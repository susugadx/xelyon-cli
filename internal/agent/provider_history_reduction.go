package agent

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ledger"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ProviderHistoryReductionMode は provider-facing history reduction の動作を表す。
type ProviderHistoryReductionMode int

const (
	// ProviderHistoryReductionDisabled は Phase 5b の no-op projection を維持する。
	ProviderHistoryReductionDisabled ProviderHistoryReductionMode = iota
	// ProviderHistoryReductionDryRun は provider payload を変えずに候補だけを記録する。
	ProviderHistoryReductionDryRun
	// ProviderHistoryReductionApply は projection clone 上で安全な候補だけを置換する。
	ProviderHistoryReductionApply
)

// ProviderHistoryReductionPolicy は provider-facing reduction の方針を選ぶ。
type ProviderHistoryReductionPolicy struct {
	Mode ProviderHistoryReductionMode
}

// ProviderHistoryReductionCandidate は dry-run detector が評価した tool result を表す。
type ProviderHistoryReductionCandidate struct {
	HistoryIndex             int
	Role                     string
	ToolName                 string
	ToolCallID               string
	OriginalByteSize         int
	OriginalRuneSize         int
	Reason                   string
	SuggestedReplacementKind string
	SuggestedReplacementText string
	KeepReason               string
	ReplacementApplied       bool
}

// ProviderHistoryProjectionReport は provider-facing projection の構築結果を要約する。
type ProviderHistoryProjectionReport struct {
	Mode                  ProviderHistoryReductionMode
	OriginalMessageCount  int
	ProjectedMessageCount int
	ToolResultCount       int
	CandidateCount        int
	KeptCount             int
	ReplacedCount         int
	OriginalBytes         int
	ProjectedBytes        int
	EstimatedSavedBytes   int
	Candidates            []ProviderHistoryReductionCandidate
	Kept                  []ProviderHistoryReductionCandidate
}

func normalizeProviderHistoryReductionPolicy(policy ProviderHistoryReductionPolicy) ProviderHistoryReductionPolicy {
	if policy.Mode != ProviderHistoryReductionDryRun && policy.Mode != ProviderHistoryReductionApply {
		policy.Mode = ProviderHistoryReductionDisabled
	}
	return policy
}

func buildProviderHistoryProjectionReport(original, projected []api.Message, policy ProviderHistoryReductionPolicy) ProviderHistoryProjectionReport {
	policy = normalizeProviderHistoryReductionPolicy(policy)
	if policy.Mode == ProviderHistoryReductionDisabled {
		return ProviderHistoryProjectionReport{}
	}

	report := buildProviderHistoryReductionDetectionReport(original, projected, policy.Mode)
	finalizeProviderHistoryProjectionReport(&report, original, projected)
	return report
}

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

func (a *Agent) applyProviderHistoryReduction(report *ProviderHistoryProjectionReport, projection []api.Message) {
	if report == nil || report.Mode != ProviderHistoryReductionApply || len(report.Candidates) == 0 {
		return
	}

	pointers := a.providerHistoryReductionEvidencePointers()
	evidenceKeyCounts := countProviderHistoryReductionEvidenceKeys(report.Candidates, report.Kept)
	for i := range report.Candidates {
		candidate := report.Candidates[i]
		key := providerHistoryReductionEvidenceKeyForCandidate(candidate)
		if evidenceKeyCounts[key] > 1 {
			keepProviderHistoryReductionCandidate(report, i, "ambiguous_evidence_pointer")
			continue
		}

		evidencePointers := providerHistoryEvidencePointersForCandidate(pointers, candidate)
		if len(evidencePointers) == 0 {
			keepProviderHistoryReductionCandidate(report, i, "missing_evidence_pointer")
			continue
		}

		if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
			keepProviderHistoryReductionCandidate(report, i, "missing_projection_message")
			continue
		}

		replacementKind, replacementText := buildProviderHistoryReplacement(candidate, evidencePointers)
		report.Candidates[i].SuggestedReplacementKind = replacementKind
		report.Candidates[i].SuggestedReplacementText = replacementText
		if len(replacementText) >= candidate.OriginalByteSize {
			keepProviderHistoryReductionCandidate(report, i, "replacement_not_smaller")
			continue
		}

		projection[candidate.HistoryIndex].Content = replacementText
		report.Candidates[i].ReplacementApplied = true
	}
}

func (a *Agent) providerHistoryReductionEvidencePointers() []ledger.EvidencePointer {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		return nil
	}
	return ledger.EvidencePointersFromState(a.Runtime.TaskLedger.Snapshot())
}

func keepProviderHistoryReductionCandidate(report *ProviderHistoryProjectionReport, candidateIndex int, reason string) {
	report.Candidates[candidateIndex].KeepReason = reason
	report.Candidates[candidateIndex].ReplacementApplied = false
	report.Kept = append(report.Kept, report.Candidates[candidateIndex])
}

type providerHistoryReductionEvidenceKey struct {
	toolCallID string
	toolName   string
}

func providerHistoryReductionEvidenceKeyForCandidate(candidate ProviderHistoryReductionCandidate) providerHistoryReductionEvidenceKey {
	return providerHistoryReductionEvidenceKey{
		toolCallID: candidate.ToolCallID,
		toolName:   candidate.ToolName,
	}
}

func countProviderHistoryReductionEvidenceKeys(entrySets ...[]ProviderHistoryReductionCandidate) map[providerHistoryReductionEvidenceKey]int {
	counts := make(map[providerHistoryReductionEvidenceKey]int)
	for _, entries := range entrySets {
		for _, entry := range entries {
			key := providerHistoryReductionEvidenceKeyForCandidate(entry)
			if key.toolCallID == "" || key.toolName == "" {
				continue
			}
			counts[key]++
		}
	}
	return counts
}

func providerHistoryEvidencePointersForCandidate(pointers []ledger.EvidencePointer, candidate ProviderHistoryReductionCandidate) []ledger.EvidencePointer {
	if len(pointers) == 0 {
		return nil
	}
	matched := make([]ledger.EvidencePointer, 0, len(pointers))
	for _, pointer := range pointers {
		if pointer.ToolCallID == candidate.ToolCallID && pointer.Source == candidate.ToolName {
			matched = append(matched, pointer)
		}
	}
	return matched
}

func buildProviderHistoryReplacement(candidate ProviderHistoryReductionCandidate, evidencePointers []ledger.EvidencePointer) (string, string) {
	toolName := providerHistoryReductionSingleLine(candidate.ToolName)
	replacementKind := providerHistoryReductionReplacementKind(toolName)
	return replacementKind, fmt.Sprintf(
		"[omitted old %s result; evidence: %s]",
		toolName,
		providerHistoryEvidencePointerSummary(evidencePointers),
	)
}

func providerHistoryReductionReplacementKind(toolName string) string {
	return fmt.Sprintf("omit_old_%s_result", providerHistoryReductionSingleLine(toolName))
}

func providerHistoryEvidencePointerSummary(evidencePointers []ledger.EvidencePointer) string {
	const maxInlineEvidencePointers = 3
	if len(evidencePointers) == 0 {
		return "missing"
	}

	limit := len(evidencePointers)
	if limit > maxInlineEvidencePointers {
		limit = maxInlineEvidencePointers
	}
	parts := make([]string, 0, limit+1)
	for _, pointer := range evidencePointers[:limit] {
		parts = append(parts, fmt.Sprintf(
			"%s:%s source=%s",
			providerHistoryReductionSingleLine(pointer.Path),
			providerHistoryEvidencePointerLineRange(pointer),
			providerHistoryReductionSingleLine(pointer.Source),
		))
	}
	if remaining := len(evidencePointers) - limit; remaining > 0 {
		parts = append(parts, fmt.Sprintf("+%d more", remaining))
	}
	return strings.Join(parts, "; ")
}

func providerHistoryEvidencePointerLineRange(pointer ledger.EvidencePointer) string {
	if pointer.EndLine > pointer.StartLine {
		return fmt.Sprintf("L%d-L%d", pointer.StartLine, pointer.EndLine)
	}
	return fmt.Sprintf("L%d", pointer.StartLine)
}

func providerHistoryReductionSingleLine(value string) string {
	value = strings.TrimSpace(value)
	value = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(value)
	if value == "" {
		return "unknown"
	}
	return value
}

func finalizeProviderHistoryProjectionReport(report *ProviderHistoryProjectionReport, original, projected []api.Message) {
	if report == nil || report.Mode == ProviderHistoryReductionDisabled {
		return
	}
	report.CandidateCount = len(report.Candidates)
	report.ReplacedCount = countProviderHistoryReplacementApplied(report.Candidates)
	report.KeptCount = report.ToolResultCount - report.ReplacedCount
	if report.KeptCount < 0 {
		report.KeptCount = 0
	}
	sort.SliceStable(report.Kept, func(i, j int) bool {
		return report.Kept[i].HistoryIndex < report.Kept[j].HistoryIndex
	})
	report.OriginalBytes = providerHistoryContentBytes(original)
	report.ProjectedBytes = providerHistoryContentBytes(projected)
	if report.OriginalBytes > report.ProjectedBytes {
		report.EstimatedSavedBytes = report.OriginalBytes - report.ProjectedBytes
	}
}

func countProviderHistoryReplacementApplied(candidates []ProviderHistoryReductionCandidate) int {
	count := 0
	for _, candidate := range candidates {
		if candidate.ReplacementApplied {
			count++
		}
	}
	return count
}

func providerHistoryContentBytes(messages []api.Message) int {
	total := 0
	for _, msg := range messages {
		total += len(msg.Content)
	}
	return total
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
