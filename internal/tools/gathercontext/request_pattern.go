package gathercontext

import "strings"

type normalizedQueryText struct {
	text              string
	literalPattern    bool
	searchRouteIntent bool
}

func normalizeQueryTextWithMetadata(query string) normalizedQueryText {
	if pattern, ok := extractQuotedPatternField(query); ok {
		return normalizedQueryText{text: pattern, literalPattern: true, searchRouteIntent: true}
	}
	if pattern, ok := extractQuotedSearchPattern(query); ok {
		return normalizedQueryText{text: pattern, literalPattern: true, searchRouteIntent: true}
	}
	return normalizedQueryText{text: query}
}

func extractQuotedPatternField(query string) (string, bool) {
	query = strings.TrimSpace(query)
	lower := strings.ToLower(query)
	if !strings.HasPrefix(lower, "pattern") || !isQueryWordBoundary(lower, len("pattern")) {
		return "", false
	}

	rest := strings.TrimSpace(query[len("pattern"):])
	if rest == "" {
		return "", false
	}
	if rest[0] == ':' || rest[0] == '=' {
		rest = strings.TrimSpace(rest[1:])
	}
	if rest == "" || (rest[0] != '"' && rest[0] != '\'') {
		return "", false
	}

	pattern, end, ok := firstQuotedSegment(rest)
	if !ok || strings.TrimSpace(rest[end:]) != "" {
		return "", false
	}
	return pattern, true
}

func extractQuotedSearchPattern(query string) (string, bool) {
	lower := strings.ToLower(query)
	patternIdx := patternWordIndex(lower)
	if patternIdx < 0 || !hasQuotedPatternSearchIntent(lower[:patternIdx]) {
		return "", false
	}

	afterPattern := query[patternIdx+len("pattern"):]
	pattern, end, ok := firstQuotedSegment(afterPattern)
	if !ok {
		return "", false
	}
	if _, _, hasAnother := firstQuotedSegment(afterPattern[end:]); hasAnother {
		return "", false
	}
	return pattern, true
}

func hasQuotedPatternSearchIntent(prefix string) bool {
	return strings.Contains(prefix, "search") ||
		strings.Contains(prefix, "find") ||
		strings.Contains(prefix, "look for")
}

func patternWordIndex(lower string) int {
	offset := 0
	for {
		idx := strings.Index(lower[offset:], "pattern")
		if idx < 0 {
			return -1
		}
		idx += offset
		end := idx + len("pattern")
		if isQueryWordBoundary(lower, idx-1) && isQueryWordBoundary(lower, end) {
			return idx
		}
		offset = end
	}
}

func isQueryWordBoundary(s string, idx int) bool {
	if idx < 0 || idx >= len(s) {
		return true
	}
	return !isQueryWordChar(s[idx])
}

func isQueryWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_'
}

func firstQuotedSegment(s string) (string, int, bool) {
	start := strings.IndexAny(s, `"'`)
	if start < 0 {
		return "", 0, false
	}
	quote := s[start]
	for i := start + 1; i < len(s); i++ {
		if s[i] == '\\' {
			i++
			continue
		}
		if s[i] == quote {
			value := strings.TrimSpace(s[start+1 : i])
			if value == "" {
				return "", 0, false
			}
			return value, i + 1, true
		}
	}
	return "", 0, false
}
