package externaldoc

import (
	"regexp"
	"strings"
)

const (
	reviewExternalDocMaxFocusTerms     = 12
	reviewExternalDocMaxFocusTermBytes = 80
)

var reviewExternalDocFocusTokenRE = regexp.MustCompile(`[A-Za-z0-9_][A-Za-z0-9_.:/-]*`)

var reviewExternalDocFocusStopwords = map[string]struct{}{
	"a":             {},
	"about":         {},
	"and":           {},
	"as":            {},
	"by":            {},
	"com":           {},
	"doc":           {},
	"docs":          {},
	"documentation": {},
	"example":       {},
	"for":           {},
	"from":          {},
	"guide":         {},
	"how":           {},
	"https":         {},
	"in":            {},
	"is":            {},
	"learn":         {},
	"of":            {},
	"on":            {},
	"overview":      {},
	"official":      {},
	"reference":     {},
	"the":           {},
	"to":            {},
	"use":           {},
	"using":         {},
	"with":          {},
	"www":           {},
}

var reviewExternalDocDocsTokens = map[string]struct{}{
	"api":       {},
	"apis":      {},
	"cache":     {},
	"caching":   {},
	"citations": {},
	"function":  {},
	"functions": {},
	"grounding": {},
	"http":      {},
	"json":      {},
	"oauth":     {},
	"request":   {},
	"response":  {},
	"responses": {},
	"sdk":       {},
	"sse":       {},
	"stream":    {},
	"streaming": {},
	"tool":      {},
	"tools":     {},
}

// BuildFocusTerms は検索 query と検索結果 metadata から snippet 抽出用 focus term を作る。
func BuildFocusTerms(query, subject, focus, resultTitle, resultSnippet string, genericTokens []string) []FocusTerm {
	builder := reviewExternalDocFocusTermBuilder{
		seen: make(map[string]struct{}),
	}
	builder.add(focus, "query focus")
	builder.add(subject, "query subject")
	for _, token := range extractReviewExternalDocSearchQueryTerms(query) {
		builder.add(token, "search query")
	}
	for _, token := range extractReviewExternalDocResultFocusTokens(resultTitle) {
		builder.add(token, "search result title")
	}
	for _, token := range extractReviewExternalDocResultFocusTokens(resultSnippet) {
		builder.add(token, "search result snippet")
	}
	for _, token := range genericTokens {
		builder.add(token, "generic impact token")
	}
	return builder.terms
}

type reviewExternalDocFocusTermBuilder struct {
	terms []FocusTerm
	seen  map[string]struct{}
}

func (b *reviewExternalDocFocusTermBuilder) add(term, reason string) {
	if len(b.terms) >= reviewExternalDocMaxFocusTerms {
		return
	}
	normalized, ok := normalizeFocusTerm(term)
	if !ok {
		return
	}
	key := strings.ToLower(normalized)
	if _, exists := b.seen[key]; exists {
		return
	}
	b.seen[key] = struct{}{}
	b.terms = append(b.terms, FocusTerm{
		Term:   normalized,
		Reason: normalizeReviewExternalDocFocusReason(reason),
	})
}

func sanitizeFocusTerms(terms []FocusTerm) []FocusTerm {
	builder := reviewExternalDocFocusTermBuilder{
		seen: make(map[string]struct{}),
	}
	for _, term := range terms {
		builder.add(term.Term, term.Reason)
	}
	return builder.terms
}

func normalizeFocusTerm(term string) (string, bool) {
	normalized := strings.Join(strings.Fields(strings.TrimSpace(term)), " ")
	normalized = strings.Trim(normalized, ".,;:")
	if normalized == "" || len(normalized) > reviewExternalDocMaxFocusTermBytes || !containsReviewExternalDocFocusAlphaNum(normalized) {
		return "", false
	}
	for _, r := range normalized {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '_' || r == '-' || r == '.' || r == '/' || r == ':' || r == ' ':
		default:
			return "", false
		}
	}
	return normalized, true
}

func normalizeReviewExternalDocFocusReason(reason string) string {
	const maxReasonBytes = 80
	normalized := strings.Join(strings.Fields(strings.TrimSpace(reason)), " ")
	if len(normalized) <= maxReasonBytes {
		return normalized
	}
	return reviewExternalDocBoundedString(normalized, maxReasonBytes)
}

func containsReviewExternalDocFocusAlphaNum(term string) bool {
	for _, r := range term {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return true
		}
	}
	return false
}

func extractReviewExternalDocSearchQueryTerms(query string) []string {
	tokens := reviewExternalDocFocusTokenRE.FindAllString(query, -1)
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !reviewExternalDocFocusTokenIsSearchTerm(token) {
			continue
		}
		result = append(result, token)
	}
	return result
}

func extractReviewExternalDocResultFocusTokens(text string) []string {
	tokens := reviewExternalDocFocusTokenRE.FindAllString(text, -1)
	result := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if !reviewExternalDocFocusTokenIsResultTerm(token) {
			continue
		}
		result = append(result, token)
	}
	return result
}

func reviewExternalDocFocusTokenIsSearchTerm(token string) bool {
	normalized, ok := normalizeFocusTerm(token)
	if !ok {
		return false
	}
	lower := strings.ToLower(normalized)
	if _, stop := reviewExternalDocFocusStopwords[lower]; stop {
		return false
	}
	return len(normalized) >= 2
}

func reviewExternalDocFocusTokenIsResultTerm(token string) bool {
	normalized, ok := normalizeFocusTerm(token)
	if !ok {
		return false
	}
	lower := strings.ToLower(normalized)
	if _, stop := reviewExternalDocFocusStopwords[lower]; stop {
		return false
	}
	if len(normalized) < 2 || strings.Contains(lower, "://") {
		return false
	}
	if _, docsToken := reviewExternalDocDocsTokens[lower]; docsToken {
		return true
	}
	return strings.ContainsAny(normalized, "_-./:") || containsReviewExternalDocDigit(normalized) || containsReviewExternalDocCamelBoundary(normalized)
}

func containsReviewExternalDocDigit(token string) bool {
	for _, r := range token {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsReviewExternalDocCamelBoundary(token string) bool {
	var prevLower bool
	for _, r := range token {
		currentUpper := r >= 'A' && r <= 'Z'
		if prevLower && currentUpper {
			return true
		}
		prevLower = r >= 'a' && r <= 'z'
	}
	return false
}
