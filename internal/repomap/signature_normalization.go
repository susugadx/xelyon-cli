package repomap

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func normalizeSignature(line string) string {
	sig := strings.TrimSpace(line)
	sig = strings.TrimSuffix(sig, "{}")
	sig = strings.TrimSuffix(sig, " {}")
	sig = strings.TrimSuffix(sig, "{")
	sig = strings.TrimSuffix(sig, " {")
	sig = strings.TrimSpace(sig)
	return sig
}

func isExportedName(name string) bool {
	r, _ := utf8.DecodeRuneInString(name)
	if r == utf8.RuneError {
		return false
	}
	return unicode.IsUpper(r)
}
