package termtext

import "strings"

// SanitizeSingleLineANSI は ANSI escape を保持しつつ単一行表示用に制御文字を正規化する。
func SanitizeSingleLineANSI(s string) string {
	if s == "" {
		return ""
	}

	var b strings.Builder
	inEscape := false
	spacePending := false

	emitSpace := func() {
		if b.Len() == 0 || spacePending {
			return
		}
		b.WriteByte(' ')
		spacePending = true
	}

	for _, r := range s {
		if inEscape {
			b.WriteRune(r)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inEscape = false
			}
			continue
		}

		if r == '\033' {
			inEscape = true
			b.WriteRune(r)
			continue
		}

		switch r {
		case '\r', '\n', '\t':
			emitSpace()
		default:
			if (r >= 0 && r < 0x20) || r == 0x7f {
				continue
			}
			b.WriteRune(r)
			spacePending = false
		}
	}

	return strings.TrimRight(b.String(), " ")
}
