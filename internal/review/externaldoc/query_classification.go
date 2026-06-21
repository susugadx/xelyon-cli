package externaldoc

import "strings"

func buildSearchQueryText(subject, focus string, intent QueryIntent) string {
	subject = strings.TrimSpace(subject)
	focus = strings.TrimSpace(focus)
	switch intent {
	case QueryIntentSpec:
		return subject + " " + focus + " specification"
	case QueryIntentSecurityAdvisory:
		return subject + " " + focus + " security advisory"
	case QueryIntentFrameworkBehavior:
		return subject + " " + focus + " documentation"
	case QueryIntentFallback:
		return subject + " " + focus + " reference"
	default:
		return subject + " " + focus + " official documentation"
	}
}

func classifySearchQueryIntent(subject, focus, corpus string) QueryIntent {
	lowerSubject := strings.ToLower(subject)
	lowerFocus := strings.ToLower(focus)
	switch {
	case strings.Contains(lowerSubject, "oauth") || strings.Contains(lowerSubject, "model context protocol"):
		return QueryIntentSpec
	case searchQuerySecuritySignal(lowerFocus):
		return QueryIntentSecurityAdvisory
	case strings.Contains(lowerSubject, "filepath") || strings.Contains(lowerSubject, "cloudflare workers"):
		return QueryIntentFrameworkBehavior
	case strings.Contains(lowerSubject, "api"):
		return QueryIntentAPIDocs
	case strings.Contains(corpus, "official") || strings.Contains(corpus, "docs"):
		return QueryIntentOfficialDocs
	default:
		return QueryIntentFallback
	}
}

func expectedSourceTypeForIntent(intent QueryIntent) QueryExpectedSourceType {
	switch intent {
	case QueryIntentAPIDocs:
		return QueryExpectedSourceAPIReference
	case QueryIntentSecurityAdvisory:
		return QueryExpectedSourceSecurityAdvisory
	case QueryIntentSpec:
		return QueryExpectedSourceTechnicalSpecification
	case QueryIntentFrameworkBehavior:
		return QueryExpectedSourceFrameworkDocumentation
	case QueryIntentFallback:
		return QueryExpectedSourceGeneralReference
	default:
		return QueryExpectedSourceOfficialDocumentation
	}
}

func searchQueryConfidenceForSubjectFocus(subject, focus string) QueryConfidence {
	if searchQueryGenericFocusTokenIsConcrete(focus) || searchQueryPreferredFocusTokenIsAllowed(focus) {
		if classifySearchQueryIntent(subject, focus, "") == QueryIntentFallback {
			return QueryConfidenceLow
		}
		return QueryConfidenceHigh
	}
	return QueryConfidenceLow
}

func searchQueryCorpusHasGoFilepathSignal(corpus string) bool {
	return strings.Contains(corpus, "path/filepath") ||
		strings.Contains(corpus, "filepath.") ||
		strings.Contains(corpus, " filepath ") ||
		strings.Contains(corpus, "evalsymlinks")
}

func searchQuerySecuritySignal(value string) bool {
	lower := strings.ToLower(strings.Join(strings.Fields(value), " "))
	if lower == "symlink" {
		return true
	}
	for _, signal := range []string{"security", "advisory", "vulnerability", "cve", "traversal"} {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

func normalizeSearchQueryText(query string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(query)), " ")
}

func normalizeSearchQueryDedupeKey(query string) string {
	return strings.ToLower(strings.ReplaceAll(normalizeSearchQueryText(query), "_", " "))
}
