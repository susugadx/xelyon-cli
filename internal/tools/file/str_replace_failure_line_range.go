package file

import "fmt"

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

func buildAppliedLineRangeStrReplaceResult(path string, plan lineRangeReplacementPlan) string {
	return fmt.Sprintf("Successfully replaced lines %d-%d in %s (new range: %d-%d)", plan.startLine, plan.endLine, path, plan.startLine, plan.replacedEndLine())
}
