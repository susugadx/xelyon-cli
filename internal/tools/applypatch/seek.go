package applypatch

import "strings"

// SeekSequence はファイル内の lines から pattern を検索する。
// start 以降で最初にマッチした位置のインデックスを返す。
// eof が true の場合、ファイル末尾からの検索を優先する。
func SeekSequence(lines []string, pattern []string, start int, eof bool) (int, bool) {
	if start < 0 {
		start = 0
	}
	if len(pattern) == 0 {
		return start, true
	}
	if len(pattern) > len(lines) {
		return 0, false
	}

	searchStart := start
	if eof && len(lines) >= len(pattern) {
		searchStart = len(lines) - len(pattern)
	}

	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool { return a == b }); ok {
		return idx, true
	}
	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool {
		return strings.TrimRight(a, " \t") == strings.TrimRight(b, " \t")
	}); ok {
		return idx, true
	}
	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool {
		return strings.TrimSpace(a) == strings.TrimSpace(b)
	}); ok {
		return idx, true
	}
	if idx, ok := seekWith(lines, pattern, searchStart, func(a, b string) bool {
		return normalizeUnicode(a) == normalizeUnicode(b)
	}); ok {
		return idx, true
	}

	return 0, false
}

func seekWith(lines, pattern []string, start int, eq func(string, string) bool) (int, bool) {
	for i := start; i <= len(lines)-len(pattern); i++ {
		match := true
		for j, p := range pattern {
			if !eq(lines[i+j], p) {
				match = false
				break
			}
		}
		if match {
			return i, true
		}
	}
	return 0, false
}

func normalizeUnicode(s string) string {
	s = strings.TrimSpace(s)

	var b strings.Builder
	for _, c := range s {
		switch {
		case c == '\u2010' || c == '\u2011' || c == '\u2012' ||
			c == '\u2013' || c == '\u2014' || c == '\u2015' || c == '\u2212':
			b.WriteByte('-')
		case c == '\u2018' || c == '\u2019' || c == '\u201A' || c == '\u201B':
			b.WriteByte('\'')
		case c == '\u201C' || c == '\u201D' || c == '\u201E' || c == '\u201F':
			b.WriteByte('"')
		case c == '\u00A0' || (c >= '\u2002' && c <= '\u200A') ||
			c == '\u202F' || c == '\u205F' || c == '\u3000':
			b.WriteByte(' ')
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}
