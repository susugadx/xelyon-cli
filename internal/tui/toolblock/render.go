package toolblock

import "strings"

// SummaryLine は tool result block の summary 行を生成する。
func SummaryLine(summary string, collapsed bool, focused bool) string {
	indicator := " "
	if focused {
		indicator = "→"
	}

	prefix := "▶"
	if !collapsed {
		prefix = "▼"
	}

	return indicator + prefix + " " + summary
}

// Lines は tool result block の表示行を生成する。
func Lines(summary string, detail string, collapsed bool, focused bool) []string {
	summaryLine := SummaryLine(summary, collapsed, focused)
	if collapsed {
		return []string{summaryLine}
	}

	lines := []string{summaryLine}
	for _, line := range strings.Split(detail, "\n") {
		lines = append(lines, "  "+line)
	}
	return lines
}
