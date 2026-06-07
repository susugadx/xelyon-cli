package providerhistory

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandoutputs"
	"github.com/susugadx/xelyon-cli/internal/providerhistory/editargs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	providerHistoryCommandEditReplacementStatusNotImplemented = providerHistoryReplacementStatusNotImplemented
	providerHistoryCommandEditReplacementStatusPartialApply   = providerHistoryReplacementStatusPartialApply
	providerHistoryCommandReplacementMinSavedTokens           = 128
	providerHistoryEditArgReplacementMinSavedTokens           = 128
)

func buildCommandEditDryRunReport(original, projection []api.Message, policy Policy, assistantToolCallsByID map[string][]providerHistoryAssistantToolCallRef, trailingToolStart, latestToolResultIndex int) CommandEditDryRunReport {
	report := newCommandEditDryRunReport()
	mode := policy.Mode
	if len(original) == 0 {
		return report
	}

	for i, msg := range original {
		if msg.Role != "tool" {
			continue
		}
		linkage := resolveProviderHistoryToolResultLinkage(original, i, msg, assistantToolCallsByID, trailingToolStart, latestToolResultIndex)
		entry := providerHistoryCommandEditDryRunEntry(i, msg, linkage.ToolName)
		if linkage.KeepReason != "" {
			if providerHistoryCommandEditLinkageHasTool(linkage) {
				entry.KeepReason = linkage.KeepReason
				report.Kept = append(report.Kept, entry)
			}
			continue
		}
		if !providerHistoryIsCommandEditTool(linkage.ToolName) {
			continue
		}
		if !providerHistoryHasLaterAssistant(original, i) {
			entry.KeepReason = "no_later_assistant_message"
			report.Kept = append(report.Kept, entry)
			continue
		}

		if providerHistoryIsCommandOutputTool(linkage.ToolName) {
			candidateIndex, ok := recordProviderHistoryCommandCandidate(&report, policy, entry, linkage.Ref.arguments, msg.Content)
			if ok && mode == DryRun {
				recordProviderHistoryCommandReplacementClassifier(&report, report.Candidates[candidateIndex])
			}
			if ok && mode == Apply {
				if report.Candidates[candidateIndex].ArtifactBackedCandidate {
					applyProviderHistoryArtifactBackedCommandReplacementCandidate(&report, policy, candidateIndex, projection)
				} else {
					applyProviderHistoryCommandReplacementCandidate(&report, candidateIndex, projection)
				}
			}
			continue
		}
		candidateIndex, ok := recordProviderHistoryEditArgCandidate(&report, entry, linkage.ToolName, linkage.Ref.arguments, msg.Content)
		if ok && mode == Apply {
			applyProviderHistoryEditArgReplacementCandidate(&report, candidateIndex, linkage.ToolName, linkage.Ref, msg.Content, projection)
		}
	}

	finalizeCommandEditDryRunReport(&report)
	return report
}

func newCommandEditDryRunReport() CommandEditDryRunReport {
	return CommandEditDryRunReport{
		ReplacementStatus: providerHistoryCommandEditReplacementStatusNotImplemented,
	}
}

func providerHistoryCommandEditDryRunEntry(historyIndex int, msg api.Message, toolName string) CommandEditDryRunCandidate {
	if toolName == "" {
		toolName = msg.ToolName
	}
	return CommandEditDryRunCandidate{
		HistoryIndex: historyIndex,
		Role:         msg.Role,
		ToolName:     toolName,
		ToolCallID:   msg.ToolCallID,
	}
}

func providerHistoryCommandEditLinkageHasTool(linkage providerHistoryToolResultLinkage) bool {
	if providerHistoryIsCommandEditTool(linkage.ToolName) {
		return true
	}
	for _, toolName := range linkage.RefToolNames {
		if providerHistoryIsCommandEditTool(toolName) {
			return true
		}
	}
	return false
}

func providerHistoryIsCommandEditTool(toolName string) bool {
	return providerHistoryIsCommandOutputTool(toolName) || providerHistoryIsEditArgTool(toolName)
}

func providerHistoryIsCommandOutputTool(toolName string) bool {
	switch toolName {
	case "bash", "command":
		return true
	default:
		return false
	}
}

func providerHistoryIsEditArgTool(toolName string) bool {
	return editargs.IsTool(toolName)
}

func recordProviderHistoryCommandCandidate(report *CommandEditDryRunReport, policy Policy, entry CommandEditDryRunCandidate, arguments, content string) (int, bool) {
	if content == "" {
		entry.Kind = "command_output"
		entry.KeepReason = "empty_command_output"
		report.Kept = append(report.Kept, entry)
		return -1, false
	}
	entry.Kind = "command_output"
	entry.OriginalByteSize = len(content)
	entry.OriginalRuneSize = utf8.RuneCountInString(content)
	entry.ApproxOriginalTokens = token.EstimateTokenCount(content)
	command := providerHistoryCommandArgument(arguments)
	decision := commandoutputs.Decide(commandoutputs.NewRequest(command, content))
	if replacement, ok := decision.Replacement(); ok {
		entry.Reason = replacement.Reason()
		entry.Classifier = replacement.Classifier()
		entry.SuggestedReplacementKind = replacement.Kind()
		entry.SuggestedReplacementText = replacement.Text()
		if savedBytes, savedTokens, ok := estimateProviderHistoryCommandReplacement(entry); ok {
			entry.EstimatedSavedBytes = savedBytes
			entry.ApproxEstimatedSavedTokens = savedTokens
			entry.ReplacementEligible = true
			report.CommandEstimatedSavedBytes += savedBytes
			report.ApproxCommandSavedTokens += savedTokens
		} else {
			entry.KeepReason = "command_replacement_below_min_saved_tokens"
			report.Kept = append(report.Kept, entry)
		}
	} else if decision.Action == commandoutputs.DecisionArtifactBackedCandidate {
		recordProviderHistoryArtifactBackedCommandCandidate(report, policy, &entry, command, content, decision)
	} else {
		keepReason := decision.KeepReason
		entry.Reason = providerHistoryCommandCandidateReasonFromKeepReason(keepReason)
		entry.KeepReason = keepReason
		report.Kept = append(report.Kept, entry)
	}
	report.Candidates = append(report.Candidates, entry)
	return len(report.Candidates) - 1, true
}

func recordProviderHistoryEditArgCandidate(report *CommandEditDryRunReport, entry CommandEditDryRunCandidate, toolName, arguments, toolResultContent string) (int, bool) {
	payload, keepReason := editargs.Payload(toolName, arguments)
	entry.Kind = "edit_arguments"
	if keepReason != "" {
		entry.KeepReason = keepReason
		report.Kept = append(report.Kept, entry)
		return -1, false
	}
	entry.OriginalByteSize = payload.Bytes
	entry.OriginalRuneSize = payload.Runes
	entry.ApproxOriginalTokens = payload.Tokens
	entry.Reason = payload.Reason
	if toolName == "delete_file" {
		entry.KeepReason = "delete_file_path_kept_context"
		report.Candidates = append(report.Candidates, entry)
		report.Kept = append(report.Kept, entry)
		return len(report.Candidates) - 1, true
	}
	if replacement, ok := editargs.BuildReplacement(editargs.ReplacementRequest{
		ToolName:          toolName,
		Arguments:         arguments,
		ToolResultContent: toolResultContent,
	}); ok {
		report.EditArgEstimatedSavedBytes += replacement.SavedBytes
		report.ApproxEditArgSavedTokens += replacement.SavedTokens
	}
	report.Candidates = append(report.Candidates, entry)
	return len(report.Candidates) - 1, true
}

func estimateProviderHistoryCommandReplacement(candidate CommandEditDryRunCandidate) (int, int, bool) {
	if candidate.Kind != "command_output" || candidate.SuggestedReplacementText == "" {
		return 0, 0, false
	}
	replacementText := candidate.SuggestedReplacementText
	if len(replacementText) >= candidate.OriginalByteSize {
		return 0, 0, false
	}
	savedTokens := clampProviderHistorySavedTokens(candidate.ApproxOriginalTokens, token.EstimateTokenCount(replacementText))
	if savedTokens < providerHistoryCommandReplacementMinSavedTokens {
		return 0, 0, false
	}
	return candidate.OriginalByteSize - len(replacementText), savedTokens, true
}

func providerHistoryCommandArgumentFields(arguments string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = map[string]json.RawMessage{}
	}
	return fields, nil
}

func providerHistoryCommandJSONStringArgument(fields map[string]json.RawMessage, key string) (string, bool) {
	raw, ok := fields[key]
	if !ok {
		return "", false
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", false
	}
	return value, true
}

func providerHistoryCommandArgument(arguments string) string {
	fields, err := providerHistoryCommandArgumentFields(arguments)
	if err != nil {
		return ""
	}
	value, _ := providerHistoryCommandJSONStringArgument(fields, "command")
	return value
}

func finalizeCommandEditDryRunReport(report *CommandEditDryRunReport) {
	if report == nil {
		return
	}
	if report.CommandReplacedCount > 0 || report.EditArgReplacedCount > 0 || report.ArtifactBackedCommandReplacedCount > 0 {
		report.ReplacementStatus = providerHistoryCommandEditReplacementStatusPartialApply
	}
	for _, candidate := range report.Candidates {
		if candidate.Reason != "" {
			if report.CandidateReasonCounts == nil {
				report.CandidateReasonCounts = make(map[string]int)
			}
			report.CandidateReasonCounts[candidate.Reason]++
		}
		switch candidate.Kind {
		case "command_output":
			report.CommandCandidates++
			report.CommandOriginalBytes += candidate.OriginalByteSize
		case "edit_arguments":
			report.EditArgCandidates++
			report.EditArgOriginalBytes += candidate.OriginalByteSize
		}
	}
	for _, kept := range report.Kept {
		if kept.KeepReason == "" {
			continue
		}
		if report.KeptReasonCounts == nil {
			report.KeptReasonCounts = make(map[string]int)
		}
		report.KeptReasonCounts[kept.KeepReason]++
	}
}

func clampProviderHistorySavedTokens(originalTokens, placeholderTokens int) int {
	if originalTokens <= placeholderTokens {
		return 0
	}
	return originalTokens - placeholderTokens
}

func applyProviderHistoryCommandReplacementCandidate(report *CommandEditDryRunReport, candidateIndex int, projection []api.Message) {
	if report == nil || candidateIndex < 0 || candidateIndex >= len(report.Candidates) {
		return
	}
	candidate := report.Candidates[candidateIndex]
	if !candidate.ReplacementEligible || candidate.SuggestedReplacementText == "" {
		return
	}
	if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
		return
	}
	if !providerHistoryCommandProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
		return
	}

	replacementText := candidate.SuggestedReplacementText
	if len(replacementText) >= candidate.OriginalByteSize {
		return
	}
	savedTokens := clampProviderHistorySavedTokens(candidate.ApproxOriginalTokens, token.EstimateTokenCount(replacementText))
	if savedTokens < providerHistoryCommandReplacementMinSavedTokens {
		return
	}

	applyProviderHistoryCommandReplacementProjection(&projection[candidate.HistoryIndex], candidate, replacementText)
	report.Candidates[candidateIndex].ReplacementApplied = true
	report.CommandReplacedCount++
	report.CommandReplacementSavedBytes += candidate.OriginalByteSize - len(replacementText)
	report.ApproxCommandReplacementSavedTokens += savedTokens
	recordProviderHistoryCommandReplacementClassifier(report, candidate)
	report.ReplacementStatus = providerHistoryCommandEditReplacementStatusPartialApply
}

func providerHistoryCommandProjectionMessageMatchesCandidate(msg api.Message, candidate CommandEditDryRunCandidate) bool {
	return msg.Role == "tool" && msg.ToolCallID == candidate.ToolCallID
}

func applyProviderHistoryCommandReplacementProjection(msg *api.Message, candidate CommandEditDryRunCandidate, replacementText string) {
	if msg == nil {
		return
	}
	if msg.ToolName == "" {
		msg.ToolName = candidate.ToolName
	}
	msg.Content = replacementText
}

func recordProviderHistoryCommandReplacementClassifier(report *CommandEditDryRunReport, candidate CommandEditDryRunCandidate) {
	if report == nil || candidate.Kind != "command_output" || !candidate.ReplacementEligible || candidate.Classifier == "" {
		return
	}
	if report.CommandReplacementClassifierCounts == nil {
		report.CommandReplacementClassifierCounts = make(map[string]int)
	}
	report.CommandReplacementClassifierCounts[candidate.Classifier]++
}

func providerHistoryCommandCandidateReasonFromKeepReason(keepReason string) string {
	switch keepReason {
	case "validation_success_without_evidence":
		return "command_output_unknown_skip"
	}
	for _, suffix := range []string{"_not_large", "_unparseable"} {
		if strings.HasSuffix(keepReason, suffix) {
			return strings.TrimSuffix(keepReason, suffix)
		}
	}
	return keepReason
}
