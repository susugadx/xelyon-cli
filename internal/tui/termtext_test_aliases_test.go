package tui

import termtext "github.com/susugadx/xelyon-cli/internal/tui/termtext"

func stripANSI(s string) string {
	return termtext.StripANSI(s)
}

func fillANSITextWidth(line string, width int, bg string) string {
	return termtext.FillANSITextWidth(line, width, bg)
}

func stylePlainTextRange(s string, startCol, endCol int, bg string) string {
	return termtext.StylePlainTextRange(s, startCol, endCol, bg)
}
