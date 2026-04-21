package file

import (
	"fmt"
	"strings"
)

func joinFailureResult(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "\n")
}

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

func buildLineRangeReplacementFailure(path string, failure lineRangeReplacementFailure) string {
	switch failure.reason {
	case lineRangeReplacementFailureMissingRange:
		return "Error: old_str is required (or provide both start_line and end_line for line-range replacement)"
	case lineRangeReplacementFailureIncompleteRange:
		return "Error: both start_line and end_line are required for line-range replacement (1-indexed inclusive)"
	case lineRangeReplacementFailureInvalidRange:
		return joinFailureResult(
			fmt.Sprintf("Error: invalid line range in %s: %v", path, failure.parseErr),
			"Next: use read_file to confirm start_line/end_line (1-indexed inclusive).",
		)
	case lineRangeReplacementFailureEmptyFile:
		return fmt.Sprintf("Error: file is empty: %s", path)
	case lineRangeReplacementFailureStartOutOfRange:
		return joinFailureResult(
			fmt.Sprintf("Error: start_line is out of range in %s (start_line=%d, file_lines=%d).", path, failure.startLine, failure.fileLines),
			"Next: use read_file to confirm the target range.",
		)
	case lineRangeReplacementFailureEndOutOfRange:
		return joinFailureResult(
			fmt.Sprintf("Error: end_line is out of range in %s (end_line=%d, file_lines=%d).", path, failure.endLine, failure.fileLines),
			"Next: use read_file to confirm the target range.",
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

func buildAppliedLineRangeStrReplaceResult(path string, plan lineRangeReplacementPlan) string {
	return fmt.Sprintf("Successfully replaced lines %d-%d in %s (new range: %d-%d)", plan.startLine, plan.endLine, path, plan.startLine, plan.replacedEndLine())
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

func buildCandidateSummary(lines []string, cands []lineRange, total int) string {
	if total <= 0 {
		return ""
	}
	shown := min(len(cands), maxFailureCandidatesToShow)
	var b strings.Builder
	fmt.Fprintf(&b, "Candidates: %d total", total)
	if shown > 0 && shown < total {
		fmt.Fprintf(&b, " (showing %d)", shown)
	}
	for i := 0; i < shown; i++ {
		c := cands[i]
		fmt.Fprintf(&b, "\n- lines %d-%d: %s", c.StartLine, c.EndLine, buildInlinePreview(lines, c.StartLine, c.EndLine, 1))
	}
	if total > shown {
		fmt.Fprintf(&b, "\n- ... %d more candidates", total-shown)
	}
	return b.String()
}

func buildHeadPreview(lines []string, limit int) string {
	if len(lines) == 0 || limit <= 0 {
		return ""
	}
	previewCount := min(limit, len(lines))
	parts := make([]string, 0, previewCount)
	for i := 0; i < previewCount; i++ {
		parts = append(parts, fmt.Sprintf("%d:%s", i+1, compactPreviewLine(lines[i])))
	}
	preview := "Preview: " + strings.Join(parts, " | ")
	if len(lines) > previewCount {
		preview += fmt.Sprintf(" | ... +%d more lines", len(lines)-previewCount)
	}
	return preview
}

func buildInlinePreview(lines []string, startLine, endLine, ctx int) string {
	if len(lines) == 0 {
		return ""
	}
	if ctx < 0 {
		ctx = 0
	}
	start := startLine - ctx
	if start < 1 {
		start = 1
	}
	end := endLine + ctx
	if end > len(lines) {
		end = len(lines)
	}
	parts := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		marker := ""
		if i >= startLine && i <= endLine {
			marker = "*"
		}
		parts = append(parts, fmt.Sprintf("%d%s:%s", i, marker, compactPreviewLine(lines[i-1])))
	}
	return strings.Join(parts, " | ")
}

func compactPreviewLine(line string) string {
	line = strings.ReplaceAll(line, "\t", " ")
	line = strings.TrimSpace(line)
	if line == "" {
		return "∅"
	}
	if len(line) <= failurePreviewLineWidth {
		return line
	}
	return line[:failurePreviewLineWidth-3] + "..."
}
