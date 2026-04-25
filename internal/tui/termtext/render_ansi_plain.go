package termtext

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/rivo/uniseg"
)

// StripANSI は ANSI escape sequence を除いた plain text を返す。
func StripANSI(s string) string {
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

// DisplayColToRuneIndex は表示列に対応する rune index を返す。
func DisplayColToRuneIndex(s string, col int) int {
	if col <= 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		next := width + PlainTextDisplayWidth(cluster)
		if col < next {
			return runeIdx
		}
		width = next
		runeIdx += utf8.RuneCountInString(cluster)
	}
	return runeIdx
}

// DisplayColToRuneIndexAfter は表示列にある grapheme cluster 直後の rune index を返す。
func DisplayColToRuneIndexAfter(s string, col int) int {
	if col < 0 {
		return 0
	}
	width := 0
	runeIdx := 0
	gr := uniseg.NewGraphemes(s)
	for gr.Next() {
		cluster := gr.Str()
		clusterRunes := utf8.RuneCountInString(cluster)
		next := width + PlainTextDisplayWidth(cluster)
		if col < next {
			return runeIdx + clusterRunes
		}
		width = next
		runeIdx += clusterRunes
	}
	return runeIdx
}

// RuneIndexToDisplayCol は rune index に対応する表示列を返す。
func RuneIndexToDisplayCol(s string, idx int) int {
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
		width += PlainTextDisplayWidth(cluster)
		runeIdx += clusterRunes
	}
	return width
}

// IsWordRune は vim motion が単語として扱う rune かどうかを返す。
func IsWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
}
