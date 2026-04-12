package tui

import "strings"

func helperSplitViewLines(view string, height int) []string {
	lines := strings.SplitN(view, "\n", height+1)
	if len(lines) > height {
		lines = lines[:height]
	}
	return lines
}
