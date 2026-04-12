package tui

import (
	"strings"
	"unicode/utf8"
)

func normalizePastedText(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	return content
}

func shouldFoldPasteBlock(content string) bool {
	content = normalizePastedText(content)
	return strings.Contains(content, "\n") || utf8.RuneCountInString(content) >= pasteBlockFoldThreshold
}

func splitRunesAt(s string, pos int) (string, string) {
	runes := []rune(s)
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	return string(runes[:pos]), string(runes[pos:])
}
