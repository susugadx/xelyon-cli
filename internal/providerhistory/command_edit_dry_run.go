package providerhistory

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/commandruntime"
	"github.com/susugadx/xelyon-cli/internal/providerhistory/editargs"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const (
	providerHistoryCommandEditReplacementStatusNotImplemented = "not_implemented"
	providerHistoryCommandEditReplacementStatusPartialApply   = "partial_apply"
	providerHistoryCommandReplacementMinSavedTokens           = 128
	providerHistoryEditArgReplacementMinSavedTokens           = 128
	providerHistoryCommandPlaceholderCommandMaxRunes          = 120
	providerHistoryCommandPlaceholder                         = "[omitted old command output; replacement not implemented]"
	providerHistoryEditArgPlaceholder                         = "[omitted old edit arguments; replacement not implemented]"
)

var (
	providerHistoryExitCodePattern          = regexp.MustCompile(`\bexit(?:ed)?(?:\s+with)?\s+(?:status|code)\s*:?\s*(-?\d+)`)
	providerHistoryGoTestSuccessLinePattern = regexp.MustCompile(`(?m)^(?:ok|\?)\s+\S+`)
	providerHistoryFailingTestLinePattern   = regexp.MustCompile(`(?mi)^\s*[1-9][0-9]*\s+failing\b`)
	providerHistoryFailedTestCountPattern   = regexp.MustCompile(`(?mi)\b[1-9][0-9]*\s+failed\b`)
	providerHistoryBuildErrorSummaryPattern = regexp.MustCompile(`(?mi)\b(?:with|has|had)\s+(?:[1-9][0-9]*\s+)?errors?\b`)
	providerHistoryLintNonzeroCountPattern  = regexp.MustCompile(`(?mi)\b[1-9][0-9]*\s+(?:errors?|issues?|problems?|warnings?)\b`)
	providerHistoryLintCleanErrorPattern    = regexp.MustCompile(`(?mi)\b0\s+errors?\b`)
	providerHistoryLintCleanProblemPattern  = regexp.MustCompile(`(?mi)\b0\s+problems?\b`)
	providerHistoryLintIssueLinePattern     = regexp.MustCompile(`(?mi)(?:^|\s)(?:errors?|issues?|problems?|warnings?)\s*:\s+\S`)
	providerHistoryLintIssueFoundPattern    = regexp.MustCompile(`(?mi)(?:^|[.;\n]\s*)(?:errors?|issues?|problems?|warnings?)\s+found\b`)
	providerHistoryLintWarningLinePattern   = regexp.MustCompile(`(?mi)\bwarning\b`)
)

var providerHistoryCommandReplacementReasonLabels = map[string]string{
	"test_success_output":  "successful test command output",
	"build_success_output": "successful build command output",
	"lint_success_output":  "successful lint command output",
}

func buildCommandEditDryRunReport(original, projection []api.Message, mode Mode, assistantToolCallsByID map[string][]providerHistoryAssistantToolCallRef, trailingToolStart, latestToolResultIndex int) CommandEditDryRunReport {
	report := newCommandEditDryRunReport()
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
			candidateIndex, ok := recordProviderHistoryCommandCandidate(&report, entry, linkage.Ref.arguments, msg.Content)
			if ok && mode == Apply {
				applyProviderHistoryCommandReplacementCandidate(&report, candidateIndex, linkage.Ref.arguments, projection)
			}
			continue
		}
		candidateIndex, ok := recordProviderHistoryEditArgCandidate(&report, entry, linkage.ToolName, linkage.Ref.arguments)
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

func recordProviderHistoryCommandCandidate(report *CommandEditDryRunReport, entry CommandEditDryRunCandidate, arguments, content string) (int, bool) {
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
	entry.Reason = classifyProviderHistoryCommandCandidateReason(arguments, content)
	report.Candidates = append(report.Candidates, entry)
	return len(report.Candidates) - 1, true
}

func recordProviderHistoryEditArgCandidate(report *CommandEditDryRunReport, entry CommandEditDryRunCandidate, toolName, arguments string) (int, bool) {
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
	report.Candidates = append(report.Candidates, entry)
	return len(report.Candidates) - 1, true
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
	commandKind := providerHistoryCommandExecutionKindFor(command)
	if commandKind != providerHistoryCommandExecutionUnknown && providerHistoryCommandOutputHasSuccessEvidence(commandKind, content) {
		return providerHistoryCommandSuccessReason(commandKind)
	}
	return "command_success_output"
}

func providerHistoryCommandArgument(arguments string) string {
	fields, err := providerHistoryCommandArgumentFields(arguments)
	if err != nil {
		return ""
	}
	value, _ := providerHistoryCommandJSONStringArgument(fields, "command")
	return value
}

func providerHistoryLooksLikeTestFailure(lowerCommand, content string) bool {
	lowerContent := strings.ToLower(content)
	commandLooksLikeTest := providerHistoryCommandExecutionKindFor(lowerCommand) == providerHistoryCommandExecutionTest
	if providerHistoryOutputHasTestFailureEvidence(content, lowerContent) {
		return true
	}
	return commandLooksLikeTest && providerHistoryLooksLikeNonzeroCommandExit(content)
}

func providerHistoryOutputHasTestFailureEvidence(content, lowerContent string) bool {
	return strings.Contains(content, "--- FAIL:") ||
		strings.Contains(content, "FAIL\t") ||
		strings.Contains(lowerContent, "test failed") ||
		strings.Contains(lowerContent, "tests failed") ||
		strings.Contains(lowerContent, "test result: failed") ||
		strings.Contains(lowerContent, "failures:") ||
		providerHistoryFailedTestCountPattern.MatchString(content) ||
		providerHistoryFailingTestLinePattern.MatchString(content)
}

func providerHistoryLooksLikeBuildFailure(lowerCommand, lowerContent string) bool {
	commandLooksLikeBuild := providerHistoryCommandExecutionKindFor(lowerCommand) == providerHistoryCommandExecutionBuild
	if providerHistoryOutputHasBuildFailureEvidence(lowerContent) {
		return true
	}
	return commandLooksLikeBuild && providerHistoryLooksLikeNonzeroCommandExit(lowerContent)
}

func providerHistoryOutputHasBuildFailureEvidence(lowerContent string) bool {
	return strings.Contains(lowerContent, "build failed") ||
		strings.Contains(lowerContent, "compile error") ||
		strings.Contains(lowerContent, "compilation error") ||
		strings.Contains(lowerContent, "undefined:") ||
		strings.Contains(lowerContent, "undeclared") ||
		providerHistoryBuildErrorSummaryPattern.MatchString(lowerContent)
}

type providerHistoryCommandExecutionKind string

const (
	providerHistoryCommandExecutionUnknown providerHistoryCommandExecutionKind = ""
	providerHistoryCommandExecutionTest    providerHistoryCommandExecutionKind = "test"
	providerHistoryCommandExecutionBuild   providerHistoryCommandExecutionKind = "build"
	providerHistoryCommandExecutionLint    providerHistoryCommandExecutionKind = "lint"
)

func providerHistoryCommandExecutionKindFor(command string) providerHistoryCommandExecutionKind {
	if providerHistoryCommandHasShellComposition(command) {
		return providerHistoryCommandExecutionUnknown
	}
	words := providerHistoryCommandWords(command)
	if len(words) == 0 {
		return providerHistoryCommandExecutionUnknown
	}
	head := providerHistoryCommandWordBase(words[0])
	second := providerHistoryCommandWordAt(words, 1)
	third := providerHistoryCommandWordAt(words, 2)

	switch {
	case head == "go" && second == "test":
		return providerHistoryCommandExecutionTest
	case head == "cargo" && second == "test":
		return providerHistoryCommandExecutionTest
	case head == "pytest":
		return providerHistoryCommandExecutionTest
	case head == "npm" && (second == "test" || second == "t" || second == "run" && third == "test"):
		return providerHistoryCommandExecutionTest
	case head == "go" && second == "build":
		return providerHistoryCommandExecutionBuild
	case head == "cargo" && second == "build":
		return providerHistoryCommandExecutionBuild
	case head == "npm" && second == "run" && third == "build":
		return providerHistoryCommandExecutionBuild
	case head == "make" && second == "build":
		return providerHistoryCommandExecutionBuild
	case head == "go" && second == "vet":
		return providerHistoryCommandExecutionLint
	case head == "golangci-lint":
		return providerHistoryCommandExecutionLint
	case head == "npm" && (second == "lint" || second == "run" && third == "lint"):
		return providerHistoryCommandExecutionLint
	case head == "eslint" || head == "ruff":
		return providerHistoryCommandExecutionLint
	case head == "npx" && (second == "eslint" || second == "ruff"):
		return providerHistoryCommandExecutionLint
	case head == "cargo" && second == "clippy":
		return providerHistoryCommandExecutionLint
	case head == "make" && second == "lint":
		return providerHistoryCommandExecutionLint
	default:
		return providerHistoryCommandExecutionUnknown
	}
}

func providerHistoryCommandWords(command string) []string {
	parts, status := commandruntime.SplitStrict(command)
	if !status.IsOK() {
		return nil
	}
	words := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}
		words = append(words, part)
	}
	for len(words) > 0 && providerHistoryLooksLikeEnvAssignment(words[0]) {
		words = words[1:]
	}
	return words
}

func providerHistoryCommandHasShellComposition(command string) bool {
	quoteChar := rune(0)
	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if quoteChar == '\'' {
			if r == '\'' {
				quoteChar = 0
			}
			continue
		}
		if quoteChar == '"' {
			if r == '\\' {
				i++
				continue
			}
			switch r {
			case '"':
				quoteChar = 0
			case '`':
				return true
			case '$':
				if i+1 < len(runes) && runes[i+1] == '(' {
					return true
				}
			}
			continue
		}
		switch r {
		case '\'', '"':
			quoteChar = r
		case '\n', '\r', ';', '|', '&', '<', '>', '`':
			return true
		case '$':
			if i+1 < len(runes) && runes[i+1] == '(' {
				return true
			}
		}
	}
	return false
}

func providerHistoryLooksLikeEnvAssignment(word string) bool {
	eq := strings.IndexByte(word, '=')
	if eq <= 0 {
		return false
	}
	for _, r := range word[:eq] {
		if r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}

func providerHistoryCommandWordAt(words []string, index int) string {
	if index < 0 || index >= len(words) {
		return ""
	}
	return providerHistoryCommandWordBase(words[index])
}

func providerHistoryCommandWordBase(word string) string {
	word = strings.TrimSpace(word)
	if word == "" {
		return ""
	}
	if idx := strings.LastIndexAny(word, `/\`); idx >= 0 && idx+1 < len(word) {
		return word[idx+1:]
	}
	return word
}

func providerHistoryCommandOutputHasSuccessEvidence(kind providerHistoryCommandExecutionKind, content string) bool {
	lowerContent := strings.ToLower(content)
	if providerHistoryLooksLikeIncompleteCommandOutput(lowerContent) || providerHistoryLooksLikeNonzeroCommandExit(content) {
		return false
	}
	switch kind {
	case providerHistoryCommandExecutionTest:
		return providerHistoryTestOutputHasSuccessEvidence(content, lowerContent) ||
			!providerHistoryOutputHasTestFailureEvidence(content, lowerContent) && providerHistoryContainsZeroExitCode(lowerContent)
	case providerHistoryCommandExecutionBuild:
		return providerHistoryBuildOutputHasSuccessEvidence(lowerContent) ||
			!providerHistoryOutputHasBuildFailureEvidence(lowerContent) && providerHistoryContainsZeroExitCode(lowerContent)
	case providerHistoryCommandExecutionLint:
		return providerHistoryLintOutputHasSuccessEvidence(lowerContent) ||
			!providerHistoryOutputHasLintIssueEvidence(lowerContent) && providerHistoryContainsZeroExitCode(lowerContent)
	default:
		return false
	}
}

func providerHistoryTestOutputHasSuccessEvidence(content, lowerContent string) bool {
	if providerHistoryOutputHasTestFailureEvidence(content, lowerContent) {
		return false
	}
	return providerHistoryGoTestSuccessLinePattern.MatchString(content) ||
		strings.Contains(lowerContent, "test result: ok") ||
		strings.Contains(lowerContent, "tests passed") ||
		strings.Contains(lowerContent, "test passed") ||
		strings.Contains(lowerContent, " passed in ") ||
		strings.Contains(lowerContent, " passing")
}

func providerHistoryBuildOutputHasSuccessEvidence(lowerContent string) bool {
	if providerHistoryOutputHasBuildFailureEvidence(lowerContent) {
		return false
	}
	return strings.Contains(lowerContent, "build completed successfully") ||
		strings.Contains(lowerContent, "build complete successfully") ||
		strings.Contains(lowerContent, "build succeeded") ||
		strings.Contains(lowerContent, "built successfully") ||
		strings.Contains(lowerContent, "successfully built") ||
		strings.Contains(lowerContent, "compiled successfully")
}

func providerHistoryLintOutputHasSuccessEvidence(lowerContent string) bool {
	if providerHistoryOutputHasLintIssueEvidence(lowerContent) {
		return false
	}
	return strings.Contains(lowerContent, "lint clean") ||
		strings.Contains(lowerContent, "lint passed") ||
		strings.Contains(lowerContent, "no lint errors") ||
		strings.Contains(lowerContent, "no issues") ||
		providerHistoryLintCleanProblemPattern.MatchString(lowerContent) ||
		providerHistoryLintCleanErrorPattern.MatchString(lowerContent)
}

func providerHistoryOutputHasLintIssueEvidence(lowerContent string) bool {
	return providerHistoryLintNonzeroCountPattern.MatchString(lowerContent) ||
		providerHistoryLintIssueLinePattern.MatchString(lowerContent) ||
		providerHistoryLintIssueFoundPattern.MatchString(lowerContent) ||
		providerHistoryLintWarningLinePattern.MatchString(lowerContent)
}

func providerHistoryLooksLikeIncompleteCommandOutput(lowerContent string) bool {
	return strings.Contains(lowerContent, "command interrupted") ||
		strings.Contains(lowerContent, "partial output") ||
		strings.Contains(lowerContent, "context canceled") ||
		strings.Contains(lowerContent, "context cancelled") ||
		strings.Contains(lowerContent, "operation canceled") ||
		strings.Contains(lowerContent, "operation cancelled") ||
		strings.Contains(lowerContent, "signal: interrupt") ||
		strings.Contains(lowerContent, "signal: killed")
}

func providerHistoryCommandSuccessReason(kind providerHistoryCommandExecutionKind) string {
	switch kind {
	case providerHistoryCommandExecutionTest:
		return "test_success_output"
	case providerHistoryCommandExecutionBuild:
		return "build_success_output"
	case providerHistoryCommandExecutionLint:
		return "lint_success_output"
	default:
		return "command_success_output"
	}
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
		exitCode, err := strconv.Atoi(match[1])
		if err == nil && exitCode != 0 {
			return true
		}
	}
	return false
}

func providerHistoryContainsZeroExitCode(lowerContent string) bool {
	matches := providerHistoryExitCodePattern.FindAllStringSubmatch(lowerContent, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		exitCode, err := strconv.Atoi(match[1])
		if err == nil && exitCode == 0 {
			return true
		}
	}
	return false
}

func finalizeCommandEditDryRunReport(report *CommandEditDryRunReport) {
	if report == nil {
		return
	}
	if report.CommandReplacedCount > 0 || report.EditArgReplacedCount > 0 {
		report.ReplacementStatus = providerHistoryCommandEditReplacementStatusPartialApply
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

func applyProviderHistoryCommandReplacementCandidate(report *CommandEditDryRunReport, candidateIndex int, arguments string, projection []api.Message) {
	if report == nil || candidateIndex < 0 || candidateIndex >= len(report.Candidates) {
		return
	}
	candidate := report.Candidates[candidateIndex]
	if !providerHistoryCommandCandidateReasonAllowsReplacement(candidate.Reason) {
		return
	}
	if candidate.HistoryIndex < 0 || candidate.HistoryIndex >= len(projection) {
		return
	}
	if !providerHistoryCommandProjectionMessageMatchesCandidate(projection[candidate.HistoryIndex], candidate) {
		return
	}

	replacementText := buildProviderHistoryCommandReplacement(candidate.Reason, providerHistoryCommandArgument(arguments))
	if len(replacementText) >= candidate.OriginalByteSize {
		return
	}
	savedTokens := clampProviderHistorySavedTokens(candidate.ApproxOriginalTokens, token.EstimateTokenCount(replacementText))
	if savedTokens < providerHistoryCommandReplacementMinSavedTokens {
		return
	}

	applyProviderHistoryCommandReplacementProjection(&projection[candidate.HistoryIndex], candidate, replacementText)
	report.CommandReplacedCount++
	report.CommandReplacementSavedBytes += candidate.OriginalByteSize - len(replacementText)
	report.ApproxCommandReplacementSavedTokens += savedTokens
	report.ReplacementStatus = providerHistoryCommandEditReplacementStatusPartialApply
}

func providerHistoryCommandCandidateReasonAllowsReplacement(reason string) bool {
	_, ok := providerHistoryCommandReplacementReasonLabels[reason]
	return ok
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

func buildProviderHistoryCommandReplacement(reason, command string) string {
	return fmt.Sprintf(
		"[omitted old %s; command=%s]",
		providerHistoryCommandReplacementReasonLabel(reason),
		providerHistoryCommandReplacementCommandSummary(command),
	)
}

func providerHistoryCommandReplacementReasonLabel(reason string) string {
	if label, ok := providerHistoryCommandReplacementReasonLabels[reason]; ok {
		return label
	}
	return "successful command output"
}

func providerHistoryCommandReplacementCommandSummary(command string) string {
	command = strings.TrimSpace(command)
	command = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ", `"`, "'").Replace(command)
	command = strings.Join(strings.Fields(command), " ")
	if command == "" {
		return "unknown"
	}
	return providerHistoryTrimRunes(command, providerHistoryCommandPlaceholderCommandMaxRunes)
}

func providerHistoryTrimRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}
	runes := []rune(value)
	if limit <= 3 {
		return string(runes[:limit])
	}
	return string(runes[:limit-3]) + "..."
}
