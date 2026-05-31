package externaldoc

import "strings"

const defaultWebSearchQueryCandidateCap = 9

// BuildSearchQueryCandidates は external doc 検索用の focused official documentation query 候補を作る。
func BuildSearchQueryCandidates(input SearchQueryPlanningInput) []SearchQueryCandidate {
	corpus := searchQueryPlanningCorpus(input)
	externalSubjects := SearchSubjectsForCorpus(corpus)
	if len(externalSubjects) == 0 {
		return nil
	}
	focusTokens := searchQueryFocusTokens(input.GenericImpactTokens, corpus)
	if len(focusTokens) == 0 {
		return nil
	}

	candidates := make([]SearchQueryCandidate, 0, len(externalSubjects)*len(focusTokens))
	seen := make(map[string]struct{})
	for _, subject := range externalSubjects {
		for _, focus := range focusTokens {
			query := strings.TrimSpace(subject + " " + focus + " official documentation")
			key := strings.ToLower(query)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, SearchQueryCandidate{
				Query:   query,
				Reason:  "changed external contract token: " + subject + " / " + focus,
				Subject: subject,
				Focus:   focus,
			})
			if len(candidates) >= defaultWebSearchQueryCandidateCap {
				return candidates
			}
		}
	}
	return candidates
}

// BuildFetchRequest は検索候補と検索結果を external doc fetch request に写像する。
func BuildFetchRequest(candidate SearchQueryCandidate, result WebSearchEvidenceResult, genericTokens []string, docID string) FetchRequest {
	return FetchRequest{
		URL:               result.URL,
		DocID:             docID,
		FocusTerms:        BuildFocusTerms(candidate.Query, candidate.Subject, candidate.Focus, result.Title, result.Snippet, genericTokens),
		SearchResultTitle: result.Title,
		QuerySubjectHint:  candidate.Subject,
	}
}

func searchQueryPlanningCorpus(input SearchQueryPlanningInput) string {
	parts := make([]string, 0, len(input.CorpusParts)+len(input.GenericImpactTokens))
	parts = append(parts, input.CorpusParts...)
	parts = append(parts, input.GenericImpactTokens...)
	return strings.ToLower(strings.Join(parts, "\n"))
}

func searchQueryFocusTokens(genericTokens []string, corpus string) []string {
	preferred := []string{
		"web_search",
		"responses API",
		"previous_response_id",
		"function calling",
		"tool_choice",
		"tool calls",
		"service_tier",
		"anthropic_version",
		"cache_control",
		"text/event-stream",
		"JSON schema",
	}
	var result []string
	seen := make(map[string]struct{})
	for _, token := range preferred {
		if !strings.Contains(corpus, strings.ToLower(token)) {
			continue
		}
		key := strings.ToLower(token)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, token)
		if len(result) >= 3 {
			return result
		}
	}
	for _, token := range genericTokens {
		token = strings.TrimSpace(token)
		if !searchQueryGenericFocusTokenIsConcrete(token) {
			continue
		}
		key := strings.ToLower(token)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, token)
		if len(result) >= 3 {
			return result
		}
	}
	return result
}

func searchQueryGenericFocusTokenIsConcrete(token string) bool {
	normalized, ok := normalizeFocusTerm(token)
	if !ok {
		return false
	}
	lower := strings.ToLower(normalized)
	switch lower {
	case "api", "apis", "config", "configuration", "provider", "providers", "model", "models", "request", "requests", "response", "responses", "streaming":
		return false
	}
	return strings.ContainsAny(normalized, "_-./:") || containsReviewExternalDocDigit(normalized) || containsReviewExternalDocCamelBoundary(normalized)
}
