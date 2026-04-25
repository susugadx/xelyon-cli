package termtext

import (
	"strings"
)

// TruncateWithANSI は ANSI エスケープを保持しつつ表示幅を制限する。
// grapheme cluster 単位で処理し、lipgloss.Width と同一基準で幅を計算する。
func TruncateWithANSI(s string, maxWidth int) string {
	var result strings.Builder
	width := 0
	inEscape := false
	i := 0
	runes := []rune(s)
	for i < len(runes) {
		r := runes[i]
		if r == '\033' {
			inEscape = true
			result.WriteRune(r)
			i++
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			i++
			continue
		}

		clusterEnd := i + 1
		for clusterEnd < len(runes) && runes[clusterEnd] != '\033' && isContinuationRune(runes[clusterEnd]) {
			clusterEnd++
		}
		cluster := string(runes[i:clusterEnd])
		w := PlainTextDisplayWidth(cluster)
		if width+w > maxWidth {
			break
		}
		result.WriteString(cluster)
		width += w
		i = clusterEnd
	}
	truncated := result.String()
	if strings.Contains(truncated, "\033[") && !strings.HasSuffix(truncated, "\033[0m") {
		truncated += "\033[0m"
	}
	return truncated
}

// isContinuationRune は grapheme cluster の続き rune かどうかを判定する。
func isContinuationRune(r rune) bool {
	switch {
	case r >= 0xFE00 && r <= 0xFE0F:
		return true
	case r >= 0x0300 && r <= 0x036F:
		return true
	case r == 0x200D:
		return true
	case r >= 0x1F1E0 && r <= 0x1F1FF:
		return true
	case r >= 0x1F3FB && r <= 0x1F3FF:
		return true
	case r >= 0x20D0 && r <= 0x20FF:
		return true
	default:
		return false
	}
}
