package tui

import termtext "github.com/susugadx/xelyon-cli/internal/tui/termtext"

func decorateViewportLine(line string, width int, bg string) string {
	return termtext.FillANSITextWidth(line, width, bg)
}

func fitANSITextWidth(line string, width int) string {
	return termtext.FillANSITextWidth(line, width, "")
}
