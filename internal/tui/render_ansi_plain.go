package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func displayColToRuneIndex(s string, col int) int {
	if col <= 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		next := width + plainTextDisplayWidth(cluster)
		if col < next {
			return runeIdx
		}
		width = next
		runeIdx += utf8.RuneCountInString(cluster)
	}
	return runeIdx
}

func displayColToRuneIndexAfter(s string, col int) int {
	if col < 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		clusterRunes := utf8.RuneCountInString(cluster)
		next := width + plainTextDisplayWidth(cluster)
		if col < next {
			return runeIdx + clusterRunes
		}
		width = next
		runeIdx += clusterRunes
	}
	return runeIdx
}

func runeIndexToDisplayCol(s string, idx int) int {
	if idx <= 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		clusterRunes := utf8.RuneCountInString(cluster)
		if runeIdx >= idx {
			break
		}
		width += plainTextDisplayWidth(cluster)
		runeIdx += clusterRunes
	}
	return width
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
