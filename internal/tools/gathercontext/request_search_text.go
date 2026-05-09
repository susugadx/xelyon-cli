package gathercontext

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/filequery"
)

func stripLeadingSearchIntent(query string) string {
	start, ok := filequery.LeadingSearchIntentPayloadStart(query)
	if !ok {
		return query
	}

	stripped := strings.TrimSpace(query[start:])
	if stripped == "" {
		return query
	}
	return stripped
}

func normalizeSearchPatternList(query string) string {
	if normalized, ok := normalizeOrPatternList(query); ok {
		return normalized
	}
	return normalizeSearchPatternPart(query)
}

func normalizeOrPatternList(query string) (string, bool) {
	parts, ok := splitQueryOnTopLevelWord(query, "or")
	if !ok {
		return "", false
	}
	patterns := make([]string, 0, len(parts))
	for _, part := range parts {
		part = normalizeSearchPatternPart(part)
		if part == "" {
			return "", false
		}
		patterns = append(patterns, part)
	}
	return strings.Join(patterns, ","), true
}

func normalizeSearchPatternPart(pattern string) string {
	pattern = strings.TrimSpace(pattern)
	unquoted, ok := unquoteQueryLiteral(pattern)
	if !ok {
		return pattern
	}
	return strings.TrimSpace(unquoted)
}
