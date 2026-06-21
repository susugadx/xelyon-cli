package mutation

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine"
)

func buildStringReplacementFailure(path, oldContent, oldStr string, failure replaceengine.StringFailure) string {
	lines := strings.Split(oldContent, "\n")
	switch {
	case failure.IsMultipleMatches():
		cands := replaceengine.FindAllOccurrencesLineRanges(oldContent, oldStr, maxFailureCandidatesToShow)
		return joinFailureResult(
			fmt.Sprintf("Error: old_str appears %d times in %s (must be unique).", failure.ExactCount(), path),
			buildCandidateSummary(lines, cands, failure.ExactCount()),
			"Next: use read_file on one candidate and retry with a more specific old_str; use start_line/end_line for a fixed range; use batch edits to replace all matches.",
		)
	case failure.IsNotFound():
		return joinFailureResult(
			fmt.Sprintf("Error: old_str not found in %s (tried exact and normalized matching).", path),
			buildHeadPreview(lines, maxFailurePreviewLines),
			"Next: use read_file/search_code to copy the exact text, then retry; use start_line/end_line if you already know the target range.",
		)
	default:
		return ""
	}
}

func buildBatchStringReplacementFailure(path string, failure replaceengine.BatchFailure) string {
	oldContent := failure.OldContent()
	oldStr := failure.OldStr()
	stringFailure := failure.StringFailure()
	lines := strings.Split(oldContent, "\n")
	switch {
	case stringFailure.IsMultipleMatches():
		cands := replaceengine.FindAllOccurrencesLineRanges(oldContent, oldStr, maxFailureCandidatesToShow)
		return joinFailureResult(
			fmt.Sprintf("Error: edits[%d].old_str appears %d times in %s (must be unique; batch aborted, no changes written).", failure.EditIndex(), stringFailure.ExactCount(), path),
			buildCandidateSummary(lines, cands, stringFailure.ExactCount()),
			fmt.Sprintf("Next: use read_file on one candidate and retry with a more specific edits[%d].old_str; use line-range mode for a fixed block.", failure.EditIndex()),
		)
	case stringFailure.IsNotFound():
		return joinFailureResult(
			fmt.Sprintf("Error: edits[%d].old_str not found in %s (tried exact and normalized matching; batch aborted, no changes written).", failure.EditIndex(), path),
			buildHeadPreview(lines, maxFailurePreviewLines),
			fmt.Sprintf("Next: use read_file/search_code to copy the exact text for edits[%d].old_str, then retry; split the batch if later edits depend on earlier changes.", failure.EditIndex()),
		)
	default:
		return ""
	}
}

func buildAppliedStrReplaceResult(path string, plan replaceengine.StringPlan) string {
	result := fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d)", path, plan.MatchStartLine(), plan.MatchEndLine(), plan.MatchStartLine(), plan.ReplacedEndLine())
	if plan.UsedNormalizedMatch() {
		result = fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d, used normalized whitespace matching)", path, plan.MatchStartLine(), plan.MatchEndLine(), plan.MatchStartLine(), plan.ReplacedEndLine())
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
