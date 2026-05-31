package review

import (
	"context"
	"fmt"
	"sort"
	"strings"
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

// ReviewExternalDocFetcher は検索結果 URL から external_doc snippet を取得する境界。
type ReviewExternalDocFetcher interface {
	FetchExternalDoc(context.Context, string, string) ReviewExternalDocEvidence
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
	query  string
	reason string
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
			doc := c.fetcher.FetchExternalDoc(ctx, searchResult.URL, docID)
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
	if !reviewWebSearchEvidenceHasFetchedSnippet(evidence) {
		evidence.Inconclusive = true
	}
	return evidence
}

func limitReviewWebSearchEvidenceResults(results []ReviewWebSearchEvidenceResult, maxResults int) ([]ReviewWebSearchEvidenceResult, bool) {
	truncated := false
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
		truncated = true
	}
	return append([]ReviewWebSearchEvidenceResult(nil), results...), truncated
}

func reviewWebSearchEvidenceHasFetchedSnippet(evidence ReviewWebSearchEvidence) bool {
	for _, doc := range evidence.ExternalDocs {
		if len(doc.Snippets) > 0 {
			return true
		}
	}
	return false
}

func buildReviewWebSearchEvidenceQueryCandidates(bundle ReviewEvidenceBundle) []reviewWebSearchEvidenceQueryCandidate {
	corpus := reviewWebSearchEvidenceCorpus(bundle)
	externalSubjects := reviewWebSearchEvidenceExternalSubjects(corpus)
	if len(externalSubjects) == 0 {
		return nil
	}
	focusTokens := reviewWebSearchEvidenceFocusTokens(bundle, corpus)
	if len(focusTokens) == 0 {
		focusTokens = []string{"API"}
	}

	candidates := make([]reviewWebSearchEvidenceQueryCandidate, 0, len(externalSubjects)*len(focusTokens))
	seen := make(map[string]struct{})
	for _, subject := range externalSubjects {
		for _, focus := range focusTokens {
			query := strings.TrimSpace(subject + " " + focus + " documentation")
			key := strings.ToLower(query)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, reviewWebSearchEvidenceQueryCandidate{
				query:  query,
				reason: "changed external contract token: " + subject + " / " + focus,
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

func reviewWebSearchEvidenceExternalSubjects(corpus string) []string {
	subjects := []struct {
		token   string
		subject string
	}{
		{token: "openai", subject: "OpenAI API"},
		{token: "responses", subject: "OpenAI Responses API"},
		{token: "anthropic", subject: "Anthropic API"},
		{token: "claude", subject: "Claude API"},
		{token: "gemini", subject: "Gemini API"},
		{token: "google", subject: "Google Gemini API"},
		{token: "kimi", subject: "Kimi API"},
		{token: "moonshot", subject: "Moonshot Kimi API"},
		{token: "bedrock", subject: "Amazon Bedrock API"},
		{token: "aws", subject: "AWS API"},
		{token: "azure", subject: "Azure OpenAI API"},
		{token: "groq", subject: "Groq API"},
		{token: "openrouter", subject: "OpenRouter API"},
		{token: "mcp", subject: "Model Context Protocol"},
		{token: "oauth", subject: "OAuth"},
		{token: "cloudflare", subject: "Cloudflare Workers"},
	}
	result := make([]string, 0)
	seen := make(map[string]struct{})
	for _, subject := range subjects {
		if !strings.Contains(corpus, subject.token) {
			continue
		}
		if _, exists := seen[subject.subject]; exists {
			continue
		}
		seen[subject.subject] = struct{}{}
		result = append(result, subject.subject)
	}
	sort.Strings(result)
	return result
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
		"prompt cache",
		"cache_control",
		"streaming",
		"citations",
		"grounding",
		"JSON schema",
		"configuration",
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
		if token == "" {
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
