package replaceengine

import (
	"fmt"
	"strconv"
	"strings"
)

// Edit は batch edits の 1 エントリを表す。
type Edit struct {
	OldStr string `json:"old_str"`
	NewStr string `json:"new_str"`
}

func batchEditLineStats(edits []Edit) BatchEditLineStats {
	stats := BatchEditLineStats{}
	for _, edit := range edits {
		stats.LinesRemoved += countEditLines(edit.OldStr)
		stats.LinesAdded += countEditLines(edit.NewStr)
	}
	return stats
}

func countEditLines(s string) int {
	if s == "" {
		return 0
	}
	parts := strings.Split(s, "\n")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		return len(parts) - 1
	}
	return len(parts)
}

// ParseLineRange は 1-indexed inclusive の start/end 行を parse する。
func ParseLineRange(startStr, endStr string) (start, end int, _ error) {
	start64, err := strconv.ParseInt(strings.TrimSpace(startStr), 10, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start_line: %w", err)
	}
	end64, err := strconv.ParseInt(strings.TrimSpace(endStr), 10, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end_line: %w", err)
	}
	start = int(start64)
	end = int(end64)

	if start < 1 {
		return 0, 0, fmt.Errorf("start_line must be >= 1")
	}
	if end < start {
		return 0, 0, fmt.Errorf("end_line must be >= start_line")
	}
	return start, end, nil
}

// FindAllOccurrencesLineRanges finds up to max occurrences of needle in content and returns their 1-indexed line ranges.
// This is best-effort and used only for error messaging.
func FindAllOccurrencesLineRanges(content, needle string, maxEntries int) [][2]int {
	if needle == "" || maxEntries <= 0 {
		return nil
	}

	var res [][2]int
	searchFrom := 0
	for len(res) < maxEntries {
		idx := strings.Index(content[searchFrom:], needle)
		if idx == -1 {
			break
		}
		absIdx := searchFrom + idx
		endIdx := absIdx + len(needle) - 1

		startLine := 1 + strings.Count(content[:absIdx], "\n")
		endLine := 1 + strings.Count(content[:endIdx], "\n")
		res = append(res, [2]int{startLine, endLine})

		searchFrom = endIdx + 1
		if searchFrom >= len(content) {
			break
		}
	}
	return res
}
