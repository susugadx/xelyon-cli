package tui

import "strings"

func plainRawTranscript(m Model) string {
	return stripANSI(strings.Join(m.rawLines, "\n"))
}
