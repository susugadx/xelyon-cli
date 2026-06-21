package readtool

import "fmt"

type readLineRange struct {
	startLine int
	endLine   int
}

func validateRequestedReadLineRange(startLine, endLine int) string {
	if startLine > 0 && endLine > 0 && startLine > endLine {
		return fmt.Sprintf("Error: start_line %d is greater than end_line %d", startLine, endLine)
	}
	return ""
}

func normalizeRequestedReadLineRange(startLine, endLine int) readLineRange {
	if startLine <= 0 {
		startLine = 1
	}
	if endLine <= 0 {
		endLine = startLine + MaxReadLines - 1
	}
	return readLineRange{startLine: startLine, endLine: endLine}
}

func resolveReadLineRange(totalLines, startLine, endLine int) (readLineRange, string) {
	if errResult := validateRequestedReadLineRange(startLine, endLine); errResult != "" {
		return readLineRange{}, errResult
	}

	window := normalizeRequestedReadLineRange(startLine, endLine)
	if window.startLine > totalLines {
		return readLineRange{}, fmt.Sprintf("Error: start_line %d exceeds total lines %d", window.startLine, totalLines)
	}
	if window.endLine > totalLines {
		window.endLine = totalLines
	}
	return window, ""
}
