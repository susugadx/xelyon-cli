package review

import (
	"strings"
)

// ---- Diff Helper Functions ----

// countAddedLines counts lines starting with '+' (excluding +++ header)
func countAddedLines(diff string) int {
	count := 0
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			count++
		}
	}
	return count
}

// extractAddedLines extracts lines starting with '+' (excluding +++ header)
func extractAddedLines(diff string) []string {
	var lines []string
	for _, line := range strings.Split(diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Remove the leading '+' for analysis
			lines = append(lines, strings.TrimPrefix(line, "+"))
		}
	}
	return lines
}

// trimSnippet trims a snippet to maxLines (single line case)
func trimSnippet(line string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 100
	}
	line = strings.TrimSpace(line)
	if len(line) > maxLen {
		return line[:maxLen] + "..."
	}
	return line
}
