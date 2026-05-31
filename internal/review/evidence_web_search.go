package review

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

const (
	defaultReviewWebSearchEvidenceMaxQueries         = 3
	defaultReviewWebSearchEvidenceMaxResultsPerQuery = 3
)

// ReviewWebSearchEvidenceCollectorOptions は外部 Web 検索 evidence collector の設定。
type ReviewWebSearchEvidenceCollectorOptions struct {
	Enabled            bool
	MaxQueries         int
	MaxResultsPerQuery int
	Searcher           ReviewWebSearchQueryRunner
	Fetcher            ReviewExternalDocFetcher
}

// ReviewWebSearchQueryRunner は review 用の非対話 Web 検索境界。
type ReviewWebSearchQueryRunner interface {
	SearchReviewWeb(context.Context, string, int) (ReviewWebSearchQueryResult, error)
}

// ReviewWebSearchQueryResult は検索 provider と URL 付き結果を表す。
type ReviewWebSearchQueryResult struct {
	Provider  string
	Results   []ReviewWebSearchEvidenceResult
	Truncated bool
}

// ReviewWebSearchEvidenceCollector は /review 用の外部 Web 検索 evidence を収集する。
type ReviewWebSearchEvidenceCollector struct {
	enabled            bool
	maxQueries         int
	maxResultsPerQuery int
	searcher           ReviewWebSearchQueryRunner
	fetcher            ReviewExternalDocFetcher
}

type reviewWebSearchEvidenceQueryCandidate struct {
	query   string
	reason  string
	subject string
	focus   string
}

// NewReviewWebSearchEvidenceCollector は外部 Web 検索 evidence collector を構築する。
func NewReviewWebSearchEvidenceCollector(opts ReviewWebSearchEvidenceCollectorOptions) *ReviewWebSearchEvidenceCollector {
	maxQueries := opts.MaxQueries
	if maxQueries <= 0 {
		maxQueries = defaultReviewWebSearchEvidenceMaxQueries
	}
	maxResults := opts.MaxResultsPerQuery
	if maxResults <= 0 {
		maxResults = defaultReviewWebSearchEvidenceMaxResultsPerQuery
	}
	fetcher := opts.Fetcher
	if fetcher == nil {
		fetcher = NewHTTPReviewExternalDocFetcher(nil)
	}
	return &ReviewWebSearchEvidenceCollector{
		enabled:            opts.Enabled,
		maxQueries:         maxQueries,
		maxResultsPerQuery: maxResults,
		searcher:           opts.Searcher,
		fetcher:            fetcher,
	}
}

// CollectWebSearchEvidence は current changes に紐づく外部仕様 evidence を収集する。
func (c *ReviewWebSearchEvidenceCollector) CollectWebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle) ReviewWebSearchEvidence {
	if c == nil || !c.enabled {
		return ReviewWebSearchEvidence{Enabled: false}
	}
	evidence := ReviewWebSearchEvidence{Enabled: true}
	if c.searcher == nil {
		evidence.Error = "web search evidence searcher is not configured"
		evidence.Inconclusive = true
		return evidence
	}
	if c.fetcher == nil {
		evidence.Error = "web search evidence fetcher is not configured"
		evidence.Inconclusive = true
		return evidence
	}

	candidates := buildReviewWebSearchEvidenceQueryCandidates(bundle)
	if len(candidates) == 0 {
		evidence.Inconclusive = true
		return evidence
	}
	if len(candidates) > c.maxQueries {
		evidence.Truncated = true
		candidates = candidates[:c.maxQueries]
	}

	var errors []string
	docIndex := 1
	for _, candidate := range candidates {
		queryEvidence := ReviewWebSearchEvidenceQuery{
			Query:  candidate.query,
			Reason: candidate.reason,
		}
		result, err := c.searcher.SearchReviewWeb(ctx, candidate.query, c.maxResultsPerQuery)
		if result.Provider != "" && evidence.Provider == "" {
			evidence.Provider = result.Provider
		}
		if err != nil {
			queryEvidence.Error = err.Error()
			errors = append(errors, fmt.Sprintf("%q: %v", candidate.query, err))
			evidence.Queries = append(evidence.Queries, queryEvidence)
			continue
		}
		if result.Truncated {
			evidence.Truncated = true
		}
		var resultsTruncated bool
		queryEvidence.Results, resultsTruncated = limitReviewWebSearchEvidenceResults(result.Results, c.maxResultsPerQuery)
		if resultsTruncated {
			evidence.Truncated = true
		}
		for _, searchResult := range queryEvidence.Results {
			docID := fmt.Sprintf("external-doc-%d", docIndex)
			docIndex++
			doc := c.fetcher.FetchExternalDoc(ctx, buildReviewExternalDocFetchRequest(candidate, searchResult, bundle.GenericImpactCandidates.Tokens, docID))
			if doc.Truncated {
				evidence.Truncated = true
			}
			if doc.Error != "" {
				errors = append(errors, fmt.Sprintf("%s: %s", searchResult.URL, doc.Error))
			}
			evidence.ExternalDocs = append(evidence.ExternalDocs, doc)
		}
		evidence.Queries = append(evidence.Queries, queryEvidence)
	}

	if len(errors) > 0 {
		evidence.Error = strings.Join(errors, "; ")
	}
	if !externaldoc.HasFetchedSnippet(evidence.ExternalDocs) {
		evidence.Inconclusive = true
	}
	return evidence
}

func buildReviewExternalDocFetchRequest(candidate reviewWebSearchEvidenceQueryCandidate, searchResult ReviewWebSearchEvidenceResult, genericTokens []string, docID string) ReviewExternalDocFetchRequest {
	return ReviewExternalDocFetchRequest{
		URL:               searchResult.URL,
		DocID:             docID,
		FocusTerms:        externaldoc.BuildFocusTerms(candidate.query, candidate.subject, candidate.focus, searchResult.Title, searchResult.Snippet, genericTokens),
		SearchResultTitle: searchResult.Title,
		QuerySubjectHint:  candidate.subject,
	}
}

func limitReviewWebSearchEvidenceResults(results []ReviewWebSearchEvidenceResult, maxResults int) ([]ReviewWebSearchEvidenceResult, bool) {
	truncated := false
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
		truncated = true
	}
	return append([]ReviewWebSearchEvidenceResult(nil), results...), truncated
}

func buildReviewWebSearchEvidenceQueryCandidates(bundle ReviewEvidenceBundle) []reviewWebSearchEvidenceQueryCandidate {
	corpus := reviewWebSearchEvidenceCorpus(bundle)
	externalSubjects := externaldoc.SearchSubjectsForCorpus(corpus)
	if len(externalSubjects) == 0 {
		return nil
	}
	focusTokens := reviewWebSearchEvidenceFocusTokens(bundle, corpus)
	if len(focusTokens) == 0 {
		return nil
	}

	candidates := make([]reviewWebSearchEvidenceQueryCandidate, 0, len(externalSubjects)*len(focusTokens))
	seen := make(map[string]struct{})
	for _, subject := range externalSubjects {
		for _, focus := range focusTokens {
			query := strings.TrimSpace(subject + " " + focus + " official documentation")
			key := strings.ToLower(query)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, reviewWebSearchEvidenceQueryCandidate{
				query:   query,
				reason:  "changed external contract token: " + subject + " / " + focus,
				subject: subject,
				focus:   focus,
			})
			if len(candidates) >= defaultReviewWebSearchEvidenceMaxQueries*3 {
				return candidates
			}
		}
	}
	return candidates
}

func reviewWebSearchEvidenceCorpus(bundle ReviewEvidenceBundle) string {
	var parts []string
	for _, file := range bundle.ChangedFiles {
		parts = append(parts, file.Path, file.OldPath)
	}
	parts = append(parts, bundle.Inventory.Config...)
	parts = append(parts, bundle.Inventory.Production...)
	parts = append(parts, bundle.Inventory.Tests...)
	parts = append(parts, bundle.Inventory.Docs...)
	parts = append(parts, bundle.Inventory.Generated...)
	for _, diff := range bundle.Diffs {
		parts = append(parts, diff.Stat, diff.NameStatus, diff.Diff)
	}
	parts = append(parts, bundle.GenericImpactCandidates.Tokens...)
	return strings.ToLower(strings.Join(parts, "\n"))
}

func reviewWebSearchEvidenceFocusTokens(bundle ReviewEvidenceBundle, corpus string) []string {
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
	for _, token := range bundle.GenericImpactCandidates.Tokens {
		token = strings.TrimSpace(token)
		if !reviewWebSearchEvidenceGenericFocusTokenIsConcrete(token) {
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

func reviewWebSearchEvidenceGenericFocusTokenIsConcrete(token string) bool {
	normalized, ok := normalizeReviewWebSearchEvidenceGenericFocusToken(token)
	if !ok {
		return false
	}
	lower := strings.ToLower(normalized)
	switch lower {
	case "api", "apis", "config", "configuration", "provider", "providers", "model", "models", "request", "requests", "response", "responses", "streaming":
		return false
	}
	return strings.ContainsAny(normalized, "_-./:") || containsReviewWebSearchEvidenceDigit(normalized) || containsReviewWebSearchEvidenceCamelBoundary(normalized)
}

func normalizeReviewWebSearchEvidenceGenericFocusToken(token string) (string, bool) {
	const maxTokenBytes = 80
	normalized := strings.Join(strings.Fields(strings.TrimSpace(token)), " ")
	normalized = strings.Trim(normalized, ".,;:")
	if normalized == "" || len(normalized) > maxTokenBytes || !containsReviewWebSearchEvidenceAlphaNum(normalized) {
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

func containsReviewWebSearchEvidenceAlphaNum(token string) bool {
	for _, r := range token {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' {
			return true
		}
	}
	return false
}

func containsReviewWebSearchEvidenceDigit(token string) bool {
	for _, r := range token {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

func containsReviewWebSearchEvidenceCamelBoundary(token string) bool {
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
