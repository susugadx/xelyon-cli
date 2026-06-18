package mutation

import (
	"fmt"
	"strings"
)

func buildStringReplacementFailure(path, oldContent, oldStr string, failure stringReplacementFailure) string {
	lines := strings.Split(oldContent, "\n")
	switch failure.reason {
	case stringReplacementFailureMultipleMatches:
		cands := findAllOccurrencesLineRanges(oldContent, oldStr, maxFailureCandidatesToShow)
		return joinFailureResult(
			fmt.Sprintf("Error: old_str appears %d times in %s (must be unique).", failure.exactCount, path),
			buildCandidateSummary(lines, cands, failure.exactCount),
			"Next: use read_file on one candidate and retry with a more specific old_str; use start_line/end_line for a fixed range; use batch edits to replace all matches.",
		)
	case stringReplacementFailureNotFound:
		return joinFailureResult(
			fmt.Sprintf("Error: old_str not found in %s (tried exact and normalized matching).", path),
			buildHeadPreview(lines, maxFailurePreviewLines),
			"Next: use read_file/search_code to copy the exact text, then retry; use start_line/end_line if you already know the target range.",
		)
	default:
		return ""
	}
}

func buildBatchStringReplacementFailure(path string, failure batchStringReplacementFailure) string {
	lines := strings.Split(failure.oldContent, "\n")
	switch failure.failure.reason {
	case stringReplacementFailureMultipleMatches:
		cands := findAllOccurrencesLineRanges(failure.oldContent, failure.oldStr, maxFailureCandidatesToShow)
		return joinFailureResult(
			fmt.Sprintf("Error: edits[%d].old_str appears %d times in %s (must be unique; batch aborted, no changes written).", failure.editIndex, failure.failure.exactCount, path),
			buildCandidateSummary(lines, cands, failure.failure.exactCount),
			fmt.Sprintf("Next: use read_file on one candidate and retry with a more specific edits[%d].old_str; use line-range mode for a fixed block.", failure.editIndex),
		)
	case stringReplacementFailureNotFound:
		return joinFailureResult(
			fmt.Sprintf("Error: edits[%d].old_str not found in %s (tried exact and normalized matching; batch aborted, no changes written).", failure.editIndex, path),
			buildHeadPreview(lines, maxFailurePreviewLines),
			fmt.Sprintf("Next: use read_file/search_code to copy the exact text for edits[%d].old_str, then retry; split the batch if later edits depend on earlier changes.", failure.editIndex),
		)
	default:
		return ""
	}
}

func buildAppliedStrReplaceResult(path string, plan stringReplacementPlan) string {
	result := fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d)", path, plan.matchStartLine, plan.matchEndLine, plan.matchStartLine, plan.replacedEndLine)
	if plan.usedNormalizedMatch {
		result = fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d, used normalized whitespace matching)", path, plan.matchStartLine, plan.matchEndLine, plan.matchStartLine, plan.replacedEndLine)
	}
	return result
}

func buildDeferredStrReplaceResult(status, mode, path, comment string) string {
	header := status + " str_replace"
	if mode != "" {
		header += " (" + mode + ")"
	}
	header += " not applied for " + path + "."
	if strings.TrimSpace(comment) == "" {
		return joinFailureResult(
			header,
			"Next: review with read_file before retrying; do not repeat the same replacement unchanged.",
		)
	}
	return joinFailureResult(
		header,
		"Comment: "+strings.TrimSpace(comment),
		"Next: review with read_file and retry only after user approval.",
	)
}
