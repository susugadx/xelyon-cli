package termtext

import "strings"

func isANSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

func applyANSICode(current, code string) string {
	if !strings.HasSuffix(code, "m") {
		return current
	}
	if code == "\033[0m" || code == "\033[m" {
		return ""
	}
	return current + code
}
