package prompt

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

const taskStateContinuationFileSummary = "recorded by deterministic task ledger"

// MergeTaskStateIntoSummaryContinuation は runtime task ledger の deterministic fact を continuation record に統合する。
func MergeTaskStateIntoSummaryContinuation(record SummaryContinuationRecord, state taskstate.RuntimeTaskState) SummaryContinuationRecord {
	record = normalizeSummaryContinuationRecord(record)
	if state.IsEmpty() {
		return record
	}
	if record.SchemaVersion == "xelyon.continuation.v1" {
		record.FilesChangedV1 = mergeTaskStateFileChanges(record.FilesChangedV1, state.ChangedFiles.Paths())
		record.Verification = mergeTaskStateVerification(record.Verification, state)
		record.DoNotRepeat = mergeTaskStateDoNotRepeat(record.DoNotRepeat, state)
		return normalizeSummaryContinuationRecord(record)
	}

	record.FilesChanged = mergeTaskStateLegacyFilesChanged(record.FilesChanged, state.ChangedFiles.Paths())
	record.DoNotRepeat = mergeTaskStateDoNotRepeat(record.DoNotRepeat, state)
	return normalizeSummaryContinuationRecord(record)
}

func mergeTaskStateFileChanges(existing []SummaryContinuationFileChange, changedPaths []string) []SummaryContinuationFileChange {
	if len(changedPaths) == 0 {
		return existing
	}

	existingByPath := make(map[string]SummaryContinuationFileChange, len(existing))
	for _, item := range existing {
		path := strings.TrimSpace(item.Path)
		if path == "" {
			continue
		}
		existingByPath[path] = item
	}

	seen := make(map[string]bool, len(existing)+len(changedPaths))
	merged := make([]SummaryContinuationFileChange, 0, len(existing)+len(changedPaths))
	for _, path := range changedPaths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		item := existingByPath[path]
		item.Path = path
		if strings.TrimSpace(item.Summary) == "" {
			item.Summary = taskStateContinuationFileSummary
		}
		merged = append(merged, item)
		seen[path] = true
	}
	for _, item := range existing {
		path := strings.TrimSpace(item.Path)
		if path == "" || seen[path] {
			continue
		}
		merged = append(merged, item)
		seen[path] = true
	}
	return merged
}

func mergeTaskStateLegacyFilesChanged(existing []string, changedPaths []string) []string {
	if len(changedPaths) == 0 {
		return existing
	}
	seen := make(map[string]bool, len(existing)+len(changedPaths))
	merged := make([]string, 0, len(existing)+len(changedPaths))
	for _, path := range changedPaths {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		merged = append(merged, path)
		seen[path] = true
	}
	for _, path := range existing {
		path = strings.TrimSpace(path)
		if path == "" || seen[path] {
			continue
		}
		merged = append(merged, path)
		seen[path] = true
	}
	return merged
}

func mergeTaskStateVerification(existing []SummaryContinuationVerification, state taskstate.RuntimeTaskState) []SummaryContinuationVerification {
	taskVerification := taskStateVerification(state)
	if len(taskVerification) == 0 {
		return existing
	}

	existingByCommand := make(map[string]SummaryContinuationVerification, len(existing))
	for _, item := range existing {
		command := summaryContinuationSingleLine(item.Command)
		if command == "" {
			continue
		}
		existingByCommand[command] = item
	}

	seen := make(map[string]bool, len(existing)+len(taskVerification))
	merged := make([]SummaryContinuationVerification, 0, len(existing)+len(taskVerification))
	for _, item := range taskVerification {
		command := summaryContinuationSingleLine(item.Command)
		if command == "" || seen[command] {
			continue
		}
		item.Command = command
		if existingItem := existingByCommand[command]; strings.TrimSpace(item.Summary) == "" {
			item.Summary = existingItem.Summary
		}
		merged = append(merged, item)
		seen[command] = true
	}
	for _, item := range existing {
		command := summaryContinuationSingleLine(item.Command)
		if command == "" || seen[command] {
			continue
		}
		merged = append(merged, item)
		seen[command] = true
	}
	return merged
}

func taskStateVerification(state taskstate.RuntimeTaskState) []SummaryContinuationVerification {
	failed := state.LastFailedTests.Results()
	passed := state.LastPassedTests.Results()
	out := make([]SummaryContinuationVerification, 0, len(failed)+len(passed))
	for _, result := range failed {
		out = append(out, taskStateTestVerification(result, "failed"))
	}
	for _, result := range passed {
		out = append(out, taskStateTestVerification(result, "passed"))
	}
	return out
}

func taskStateTestVerification(result taskstate.TestResult, status string) SummaryContinuationVerification {
	return SummaryContinuationVerification{
		Command: taskStateTestCommand(result),
		Status:  status,
		Summary: taskStateTestSummary(result),
	}
}

func taskStateTestCommand(result taskstate.TestResult) string {
	return summaryContinuationSingleLine(result.Command())
}

func taskStateTestSummary(result taskstate.TestResult) string {
	excerpt := summaryTaskStateExcerpt(result.Excerpt())
	if excerpt == "" {
		return "recorded by deterministic task ledger"
	}
	return "excerpt: " + excerpt
}

func mergeTaskStateDoNotRepeat(existing []string, state taskstate.RuntimeTaskState) []string {
	signatures := taskStateDoNotRepeatSignatures(state)
	supersededCommands := taskStateLedgerTestCommandSet(state)
	if len(signatures) == 0 && len(supersededCommands) == 0 {
		return existing
	}
	seen := make(map[string]bool, len(existing)+len(signatures))
	merged := make([]string, 0, len(existing)+len(signatures))
	for _, value := range existing {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if taskStateFailedTestSignatureSuperseded(value, supersededCommands) {
			continue
		}
		key := summaryTaskStateDedupeKey(value)
		if seen[key] {
			continue
		}
		merged = append(merged, value)
		seen[key] = true
	}
	for _, value := range signatures {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := summaryTaskStateDedupeKey(value)
		if seen[key] {
			continue
		}
		merged = append(merged, value)
		seen[key] = true
	}
	return merged
}

func taskStateLedgerTestCommandSet(state taskstate.RuntimeTaskState) map[string]bool {
	failed := state.LastFailedTests.Results()
	passed := state.LastPassedTests.Results()
	out := make(map[string]bool, len(failed)+len(passed))
	for _, result := range failed {
		if command := taskStateTestCommand(result); command != "" {
			out[command] = true
		}
	}
	for _, result := range passed {
		if command := taskStateTestCommand(result); command != "" {
			out[command] = true
		}
	}
	return out
}

func taskStateFailedTestSignatureSuperseded(value string, commands map[string]bool) bool {
	if len(commands) == 0 {
		return false
	}
	for command := range commands {
		if taskStateFailedTestSignatureMatchesCommand(value, command) {
			return true
		}
	}
	return false
}

func taskStateFailedTestSignatureMatchesCommand(value, command string) bool {
	const prefix = "failed test:"
	value = summaryContinuationSingleLine(value)
	command = summaryContinuationSingleLine(command)
	if value == "" || command == "" || !strings.HasPrefix(strings.ToLower(value), prefix) {
		return false
	}
	rest := strings.TrimSpace(value[len(prefix):])
	return rest == command ||
		strings.HasPrefix(rest, command+" exit=") ||
		strings.HasPrefix(rest, command+" excerpt=")
}

func taskStateDoNotRepeatSignatures(state taskstate.RuntimeTaskState) []string {
	results := state.LastFailedTests.Results()
	if len(results) == 0 {
		return nil
	}
	out := make([]string, 0, len(results))
	for _, result := range results {
		command := taskStateTestCommand(result)
		if command == "" {
			continue
		}
		signature := fmt.Sprintf("failed test: %s", command)
		if result.ExitCode() != 0 {
			signature += fmt.Sprintf(" exit=%d", result.ExitCode())
		}
		if excerpt := summaryTaskStateExcerpt(result.Excerpt()); excerpt != "" {
			signature += " excerpt=" + excerpt
		}
		out = append(out, signature)
	}
	return out
}

func summaryTaskStateExcerpt(value string) string {
	value = taskstate.FormatSnapshotExcerpt(value, taskstate.DefaultSnapshotRenderOptions().ExcerptRuneLimit)
	if value == "" {
		return ""
	}
	return value
}

func summaryTaskStateDedupeKey(value string) string {
	return strings.ToLower(summaryContinuationSingleLine(value))
}
