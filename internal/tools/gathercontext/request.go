package gathercontext

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filequery"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

type request struct {
	query                string
	path                 string
	fileFilter           string
	searchQuery          string
	searchPath           string
	searchRouteIntent    bool
	literalSearchPattern bool
}

func normalizeRequest(req request) request {
	rawQuery := strings.TrimSpace(req.query)
	normalizedQuery := normalizeQueryTextWithMetadata(rawQuery)
	path := strings.TrimSpace(req.path)

	normalized := request{
		query:                normalizedQuery.text,
		path:                 path,
		fileFilter:           strings.TrimSpace(req.fileFilter),
		searchQuery:          strings.TrimSpace(req.searchQuery),
		searchPath:           strings.TrimSpace(req.searchPath),
		searchRouteIntent:    req.searchRouteIntent || filequery.LooksLikeNaturalLanguageSearchIntent(rawQuery) || normalizedQuery.searchRouteIntent,
		literalSearchPattern: req.literalSearchPattern || normalizedQuery.literalPattern,
	}
	if normalized.searchQuery == "" {
		normalized.searchQuery = normalized.query
	}
	if normalized.searchPath == "" {
		normalized.searchPath = normalized.path
	}
	return normalizeSearchRouteFields(normalized)
}

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

func parseRequestArgs(args map[string]string) (request, string) {
	req := normalizeRequest(request{
		query:      strings.TrimSpace(args["query"]),
		path:       strings.TrimSpace(args["path"]),
		fileFilter: strings.TrimSpace(args["file_filter"]),
	})
	if req.query == "" {
		return request{}, "Error: query is required"
	}
	if !hasEffectiveSearchPattern(req) {
		return request{}, "Error: query must include at least one non-empty term"
	}
	return req, ""
}

func hasEffectiveSearchPattern(req request) bool {
	if req.literalSearchPattern {
		return strings.TrimSpace(req.searchQuery) != ""
	}
	return search.HasEffectivePatternList(req.searchQuery)
}
