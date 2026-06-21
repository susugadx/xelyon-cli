package mutation

import "fmt"

import "github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine"

func buildLineRangeReplacementFailure(path string, failure replaceengine.LineRangeFailure) string {
	switch {
	case failure.IsMissingRange():
		return "Error: old_str is required (or provide both start_line and end_line for line-range replacement)"
	case failure.IsIncompleteRange():
		return "Error: both start_line and end_line are required for line-range replacement (1-indexed inclusive)"
	case failure.IsInvalidRange():
		return joinFailureResult(
			fmt.Sprintf("Error: invalid line range in %s: %v", path, failure.ParseErr()),
			"Next: use read_file to confirm start_line/end_line (1-indexed inclusive).",
		)
	case failure.IsEmptyFile():
		return fmt.Sprintf("Error: file is empty: %s", path)
	case failure.IsStartOutOfRange():
		return joinFailureResult(
			fmt.Sprintf("Error: start_line is out of range in %s (start_line=%d, file_lines=%d).", path, failure.StartLine(), failure.FileLines()),
			"Next: use read_file to confirm the target range.",
		)
	case failure.IsEndOutOfRange():
		return joinFailureResult(
			fmt.Sprintf("Error: end_line is out of range in %s (end_line=%d, file_lines=%d).", path, failure.EndLine(), failure.FileLines()),
			"Next: use read_file to confirm the target range.",
		)
	default:
		return ""
	}
}

func buildAppliedLineRangeStrReplaceResult(path string, plan replaceengine.LineRangePlan) string {
	return fmt.Sprintf("Successfully replaced lines %d-%d in %s (new range: %d-%d)", plan.StartLine(), plan.EndLine(), path, plan.StartLine(), plan.ReplacedEndLine())
}
