package search

import (
	"fmt"
	"regexp"
	"strings"
)

// SearchQueryAnalysis はクエリ解析結果。
type SearchQueryAnalysis struct {
	Pattern                 string
	TrimmedPattern          string
	HasWhitespace           bool
	LooksLikeBareIdentifier bool
	LooksLikeDottedSymbol   bool
	StrongRegexMeta         bool
	LooksLiteralText        bool
}

func analyzeSearchQuery(pattern string) SearchQueryAnalysis {
	trimmed := strings.TrimSpace(pattern)
	looksLikeSymbol := looksLikeIdentifier(trimmed)

	analysis := SearchQueryAnalysis{
		Pattern:                 pattern,
		TrimmedPattern:          trimmed,
		HasWhitespace:           strings.ContainsAny(trimmed, " \t\r\n"),
		LooksLikeBareIdentifier: looksLikeSymbol && !strings.Contains(trimmed, ".") && !strings.HasPrefix(trimmed, "(*"),
		LooksLikeDottedSymbol:   looksLikeSymbol && !strings.EqualFold(trimmed, ""),
		StrongRegexMeta:         hasStrongRegexMeta(trimmed),
	}
	if analysis.LooksLikeBareIdentifier {
		analysis.LooksLikeDottedSymbol = false
	}
	analysis.LooksLiteralText = trimmed != "" && !analysis.StrongRegexMeta && !looksLikeSymbol
	return analysis
}

func (a SearchQueryAnalysis) defaultTextLane() searchLane {
	if a.StrongRegexMeta {
		return searchLaneRegex
	}
	return searchLaneLiteral
}

func hasStrongRegexMeta(s string) bool {
	if s == "" {
		return false
	}
	if containsRegexMeta(s) {
		return true
	}
	return strings.Contains(s, ".*") ||
		strings.Contains(s, ".+") ||
		strings.Contains(s, `\(`) ||
		strings.Contains(s, `\)`) ||
		strings.Contains(s, `\.`)
}

var (
	goReceiverMethodRescueRe = regexp.MustCompile(`func\s+\\\([^)]*\\\*?([A-Za-z_][A-Za-z0-9_]*)[^)]*\\\)\s+([A-Za-z_][A-Za-z0-9_]*)\\\(`)
	goSelectorRescueRe       = regexp.MustCompile(`(?:^|\\\.)\s*([A-Za-z_][A-Za-z0-9_]*)\\\(`)
	goCallRescueRe           = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\\\(`)
	identTokenRe             = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_]*`)
)

var goSymbolRescueStopwords = map[string]bool{
	"func": true, "type": true, "var": true, "const": true, "package": true,
	"return": true, "if": true, "for": true, "switch": true, "case": true,
}

func shouldTryGoSymbolRescue(pattern string, analysis SearchQueryAnalysis) bool {
	if analysis.TrimmedPattern == "" {
		return false
	}
	if analysis.LooksLikeBareIdentifier || analysis.LooksLikeDottedSymbol {
		return true
	}
	if !analysis.StrongRegexMeta {
		return false
	}
	return strings.Contains(pattern, `\(`) ||
		strings.Contains(pattern, `\.`) ||
		strings.Contains(pattern, "func ") ||
		strings.Contains(pattern, "(*")
}

func extractGoSymbolRescueCandidates(pattern string) []string {
	trimmed := strings.TrimSpace(pattern)
	var candidates []string
	seen := make(map[string]bool)
	add := func(candidate string) {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" || seen[candidate] {
			return
		}
		seen[candidate] = true
		candidates = append(candidates, candidate)
	}

	if looksLikeIdentifier(trimmed) {
		add(trimmed)
		if idx := strings.LastIndex(trimmed, "."); idx >= 0 && idx+1 < len(trimmed) {
			add(trimmed[idx+1:])
		}
		return candidates
	}

	if m := goReceiverMethodRescueRe.FindStringSubmatch(trimmed); len(m) == 3 {
		add(fmt.Sprintf("(*%s).%s", m[1], m[2]))
		add(m[2])
	}
	if m := goSelectorRescueRe.FindStringSubmatch(trimmed); len(m) == 2 {
		add(m[1])
	}
	if m := goCallRescueRe.FindStringSubmatch(trimmed); len(m) == 2 {
		add(m[1])
	}

	tokens := identTokenRe.FindAllString(trimmed, -1)
	for i := len(tokens) - 1; i >= 0; i-- {
		if !goSymbolRescueStopwords[tokens[i]] {
			add(tokens[i])
			break
		}
	}

	return candidates
}
