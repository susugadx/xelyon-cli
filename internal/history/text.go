package history

import "unicode/utf8"

// TruncateWithEllipsis は max runes を超える文字列を "..." 付きで切り詰める。
func TruncateWithEllipsis(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	return truncateRunes(s, maxRunes) + "..."
}
