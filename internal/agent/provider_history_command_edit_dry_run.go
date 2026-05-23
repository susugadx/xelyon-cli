package agent

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/api"
)

const (
	providerHistoryCommandEditReplacementStatusNotImplemented = "not_implemented"
	providerHistoryCommandPlaceholder                         = "[omitted old command output; replacement not implemented]"
	providerHistoryEditArgPlaceholder                         = "[omitted old edit arguments; replacement not implemented]"
)

var providerHistoryExitCodePattern = regexp.MustCompile(`\bexit(?:ed)?(?:\s+with)?\s+(?:status|code)\s*:?\s*(-?\d+)`)

func buildProviderHistoryCommandEditDryRunReport(original []api.Message, assistantToolCallsByID map[string][]providerHistoryAssistantToolCallRef, trailingToolStart, latestToolResultIndex int) ProviderHistoryCommandEditDryRunReport {
	report := newProviderHistoryCommandEditDryRunReport()
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
			recordProviderHistoryCommandCandidate(&report, entry, linkage.Ref.arguments, msg.Content)
			continue
		}
		recordProviderHistoryEditArgCandidate(&report, entry, linkage.ToolName, linkage.Ref.arguments)
	}

	finalizeProviderHistoryCommandEditDryRunReport(&report)
	return report
}

func newProviderHistoryCommandEditDryRunReport() ProviderHistoryCommandEditDryRunReport {
	return ProviderHistoryCommandEditDryRunReport{
		ReplacementStatus: providerHistoryCommandEditReplacementStatusNotImplemented,
	}
}

func providerHistoryCommandEditDryRunEntry(historyIndex int, msg api.Message, toolName string) ProviderHistoryCommandEditDryRunCandidate {
	if toolName == "" {
		toolName = msg.ToolName
	}
	return ProviderHistoryCommandEditDryRunCandidate{
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
	switch toolName {
	case "write_file", "apply_patch", "str_replace", "delete_file":
		return true
	default:
		return false
	}
}

func recordProviderHistoryCommandCandidate(report *ProviderHistoryCommandEditDryRunReport, entry ProviderHistoryCommandEditDryRunCandidate, arguments, content string) {
	if content == "" {
		entry.Kind = "command_output"
		entry.KeepReason = "empty_command_output"
		report.Kept = append(report.Kept, entry)
		return
	}
	entry.Kind = "command_output"
	entry.OriginalByteSize = len(content)
	entry.OriginalRuneSize = utf8.RuneCountInString(content)
	entry.ApproxOriginalTokens = token.EstimateTokenCount(content)
	entry.Reason = classifyProviderHistoryCommandCandidateReason(arguments, content)
	report.Candidates = append(report.Candidates, entry)
}

func recordProviderHistoryEditArgCandidate(report *ProviderHistoryCommandEditDryRunReport, entry ProviderHistoryCommandEditDryRunCandidate, toolName, arguments string) {
	payload, keepReason := providerHistoryEditArgPayload(toolName, arguments)
	entry.Kind = "edit_arguments"
	if keepReason != "" {
		entry.KeepReason = keepReason
		report.Kept = append(report.Kept, entry)
		return
	}
	entry.OriginalByteSize = payload.bytes
	entry.OriginalRuneSize = payload.runes
	entry.ApproxOriginalTokens = payload.tokens
	entry.Reason = payload.reason
	report.Candidates = append(report.Candidates, entry)
}

type providerHistoryEditArgPayloadSummary struct {
	reason string
	bytes  int
	runes  int
	tokens int
}

func providerHistoryEditArgPayload(toolName, arguments string) (providerHistoryEditArgPayloadSummary, string) {
	fields, err := providerHistoryToolCallArgumentFields(arguments)
	if err != nil {
		return providerHistoryEditArgPayloadSummary{}, "invalid_tool_call_arguments"
	}
	switch toolName {
	case "write_file":
		return providerHistoryStringFieldsPayload("write_file_content", fields, "content")
	case "apply_patch":
		return providerHistoryStringFieldsPayload("apply_patch_patch", fields, "patch")
	case "str_replace":
		if raw, ok := fields["edits"]; ok && len(raw) > 0 && string(raw) != "null" {
			return providerHistoryRawOrStringFieldPayload("str_replace_edits", raw)
		}
		return providerHistoryStringFieldsPayload("str_replace_strings", fields, "old_str", "new_str")
	case "delete_file":
		if value, ok := providerHistoryJSONStringArgument(fields, "path"); ok && value != "" {
			return providerHistoryValuesPayload("delete_file_path", []string{value}), ""
		}
		return providerHistoryEditArgPayloadSummary{}, "missing_edit_argument_payload"
	default:
		return providerHistoryEditArgPayloadSummary{}, "tool_not_in_command_edit_allowlist"
	}
}

func providerHistoryToolCallArgumentFields(arguments string) (map[string]json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &fields); err != nil {
		return nil, err
	}
	if fields == nil {
		fields = map[string]json.RawMessage{}
	}
	return fields, nil
}

func providerHistoryStringFieldsPayload(reason string, fields map[string]json.RawMessage, keys ...string) (providerHistoryEditArgPayloadSummary, string) {
	var values []string
	for _, key := range keys {
		value, ok := providerHistoryJSONStringArgument(fields, key)
		if !ok {
			continue
		}
		values = append(values, value)
	}
	if len(values) == 0 {
		return providerHistoryEditArgPayloadSummary{}, "missing_edit_argument_payload"
	}
	return providerHistoryValuesPayload(reason, values), ""
}

func providerHistoryRawOrStringFieldPayload(reason string, raw json.RawMessage) (providerHistoryEditArgPayloadSummary, string) {
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		return providerHistoryValuesPayload(reason, []string{value}), ""
	}
	rawValue := strings.TrimSpace(string(raw))
	if rawValue == "" {
		return providerHistoryEditArgPayloadSummary{}, "missing_edit_argument_payload"
	}
	return providerHistoryValuesPayload(reason, []string{rawValue}), ""
}

func providerHistoryJSONStringArgument(fields map[string]json.RawMessage, key string) (string, bool) {
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

func providerHistoryValuesPayload(reason string, values []string) providerHistoryEditArgPayloadSummary {
	summary := providerHistoryEditArgPayloadSummary{reason: reason}
	for _, value := range values {
		summary.bytes += len(value)
		summary.runes += utf8.RuneCountInString(value)
		summary.tokens += token.EstimateTokenCount(value)
	}
	return summary
}

func classifyProviderHistoryCommandCandidateReason(arguments, content string) string {
	command := providerHistoryCommandArgument(arguments)
	lowerCommand := strings.ToLower(command)
	lowerContent := strings.ToLower(content)
	if strings.Contains(lowerCommand, "git diff") || strings.Contains(lowerContent, "diff --git") {
		return "git_diff_output"
	}
	if providerHistoryLooksLikeTestFailure(lowerCommand, content) {
		return "test_failure_output"
	}
	if providerHistoryLooksLikeBuildFailure(lowerCommand, lowerContent) {
		return "build_failure_output"
	}
	if providerHistoryLooksLikeNonzeroCommandExit(content) {
		return "command_exit_nonzero"
	}
	return "command_output"
}

func providerHistoryCommandArgument(arguments string) string {
	fields, err := providerHistoryToolCallArgumentFields(arguments)
	if err != nil {
		return ""
	}
	value, _ := providerHistoryJSONStringArgument(fields, "command")
	return value
}

func providerHistoryLooksLikeTestFailure(lowerCommand, content string) bool {
	lowerContent := strings.ToLower(content)
	commandLooksLikeTest := strings.Contains(lowerCommand, "go test") ||
		strings.Contains(lowerCommand, "cargo test") ||
		strings.Contains(lowerCommand, "pytest") ||
		strings.Contains(lowerCommand, "npm test") ||
		strings.Contains(lowerCommand, "npm run test")
	outputLooksLikeTestFailure := strings.Contains(content, "--- FAIL:") ||
		strings.Contains(content, "FAIL\t") ||
		strings.Contains(lowerContent, "test failed") ||
		strings.Contains(lowerContent, "tests failed")
	if outputLooksLikeTestFailure {
		return true
	}
	return commandLooksLikeTest && providerHistoryLooksLikeNonzeroCommandExit(content)
}

func providerHistoryLooksLikeBuildFailure(lowerCommand, lowerContent string) bool {
	commandLooksLikeBuild := strings.Contains(lowerCommand, "go build") ||
		strings.Contains(lowerCommand, "cargo build") ||
		strings.Contains(lowerCommand, "npm run build") ||
		strings.Contains(lowerCommand, "make build")
	outputLooksLikeBuildFailure := strings.Contains(lowerContent, "build failed") ||
		strings.Contains(lowerContent, "compile error") ||
		strings.Contains(lowerContent, "compilation error") ||
		strings.Contains(lowerContent, "undefined:") ||
		strings.Contains(lowerContent, "undeclared")
	if outputLooksLikeBuildFailure {
		return true
	}
	return commandLooksLikeBuild && providerHistoryLooksLikeNonzeroCommandExit(lowerContent)
}

func providerHistoryLooksLikeNonzeroCommandExit(content string) bool {
	trimmed := strings.TrimSpace(content)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(trimmed, "Error:") {
		return true
	}
	if providerHistoryContainsNonzeroExitCode(lower) {
		return true
	}
	return strings.Contains(lower, "command failed") ||
		strings.Contains(lower, "non-zero")
}

func providerHistoryContainsNonzeroExitCode(lowerContent string) bool {
	matches := providerHistoryExitCodePattern.FindAllStringSubmatch(lowerContent, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		if strings.TrimLeft(match[1], "-0") != "" {
			return true
		}
	}
	return false
}

func finalizeProviderHistoryCommandEditDryRunReport(report *ProviderHistoryCommandEditDryRunReport) {
	if report == nil {
		return
	}
	commandPlaceholderTokens := token.EstimateTokenCount(providerHistoryCommandPlaceholder)
	editPlaceholderTokens := token.EstimateTokenCount(providerHistoryEditArgPlaceholder)
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
			report.ApproxCommandSavedTokens += clampProviderHistorySavedTokens(candidate.ApproxOriginalTokens, commandPlaceholderTokens)
		case "edit_arguments":
			report.EditArgCandidates++
			report.EditArgOriginalBytes += candidate.OriginalByteSize
			report.ApproxEditArgSavedTokens += clampProviderHistorySavedTokens(candidate.ApproxOriginalTokens, editPlaceholderTokens)
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
