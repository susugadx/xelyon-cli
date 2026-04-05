package file

import "fmt"

type readLineRange struct {
	startLine int
	endLine   int
}

func resolveReadLineRange(totalLines, startLine, endLine int) (readLineRange, string) {
	if startLine <= 0 {
		startLine = 1
	}
	if endLine <= 0 {
		endLine = startLine + MaxReadLines - 1
	}
	if startLine > totalLines {
		return readLineRange{}, fmt.Sprintf("Error: start_line %d exceeds total lines %d", startLine, totalLines)
	}
	if endLine > totalLines {
		endLine = totalLines
	}
	if startLine > endLine {
		return readLineRange{}, fmt.Sprintf("Error: start_line %d is greater than end_line %d", startLine, endLine)
	}
	return readLineRange{startLine: startLine, endLine: endLine}, ""
}
