package tui

import "strings"

func rebuildStreamLine(cells []streamCell) string {
	var b strings.Builder
	currentStyle := ""
	for _, cell := range cells {
		if cell.span == 0 {
			continue
		}
		if cell.style != currentStyle {
			if currentStyle != "" {
				b.WriteString("\033[0m")
			}
			if cell.style != "" {
				b.WriteString(cell.style)
			}
			currentStyle = cell.style
		}
		b.WriteString(cell.text)
	}
	if currentStyle != "" {
		b.WriteString("\033[0m")
	}
	return b.String()
}
