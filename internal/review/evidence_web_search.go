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

// ReviewWebSearchEvidenceCollector は /review 用の外部 Web 検索 evidence を収集する。
type ReviewWebSearchEvidenceCollector struct {
	enabled            bool
	maxQueries         int
	maxResultsPerQuery int
	searcher           ReviewWebSearchQueryRunner
	fetcher            ReviewExternalDocFetcher
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

	candidates := externaldoc.BuildSearchQueryCandidates(buildReviewWebSearchQueryPlanningInput(bundle))
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
			Query:  candidate.Query,
			Reason: candidate.Reason,
		}
		result, err := c.searcher.SearchReviewWeb(ctx, candidate.Query, c.maxResultsPerQuery)
		if result.Provider != "" && evidence.Provider == "" {
			evidence.Provider = result.Provider
		}
		if err != nil {
			queryEvidence.Error = err.Error()
			errors = append(errors, fmt.Sprintf("%q: %v", candidate.Query, err))
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
			doc := c.fetcher.FetchExternalDoc(ctx, externaldoc.BuildFetchRequest(candidate, searchResult, bundle.GenericImpactCandidates.Tokens, docID))
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

func limitReviewWebSearchEvidenceResults(results []ReviewWebSearchEvidenceResult, maxResults int) ([]ReviewWebSearchEvidenceResult, bool) {
	truncated := false
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
		truncated = true
	}
	return append([]ReviewWebSearchEvidenceResult(nil), results...), truncated
}

func buildReviewWebSearchQueryPlanningInput(bundle ReviewEvidenceBundle) externaldoc.SearchQueryPlanningInput {
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
	return externaldoc.SearchQueryPlanningInput{
		CorpusParts:         parts,
		GenericImpactTokens: append([]string(nil), bundle.GenericImpactCandidates.Tokens...),
	}
}
