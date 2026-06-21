package evidence

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
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
	Fetcher            externaldoc.Fetcher
}

// ReviewWebSearchQueryRunner は review 用の非対話 Web 検索境界。
type ReviewWebSearchQueryRunner interface {
	SearchReviewWeb(context.Context, string, int) (externaldoc.WebSearchQueryResult, error)
}

// ReviewWebSearchEvidenceCollector は /review 用の外部 Web 検索 evidence を収集する。
type ReviewWebSearchEvidenceCollector struct {
	enabled            bool
	maxQueries         int
	maxResultsPerQuery int
	searcher           ReviewWebSearchQueryRunner
	fetcher            externaldoc.Fetcher
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
		fetcher = externaldoc.NewHTTPFetcher(nil)
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
func (c *ReviewWebSearchEvidenceCollector) CollectWebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle) externaldoc.WebSearchEvidence {
	return c.collectWebSearchEvidence(ctx, bundle, buildReviewWebSearchQueryPlanningInput(bundle), externaldoc.WebSearchEvidence{})
}

// CollectPostPass1WebSearchEvidence は Pass1 probe plan から追加の外部仕様 evidence を収集し、既存 evidence に merge する。
func (c *ReviewWebSearchEvidenceCollector) CollectPostPass1WebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, plan reviewprobeplan.ReviewProbePlan) externaldoc.WebSearchEvidence {
	return c.collectWebSearchEvidence(ctx, bundle, buildReviewPostPass1WebSearchQueryPlanningInput(plan), bundle.WebSearchEvidence)
}

func (c *ReviewWebSearchEvidenceCollector) collectWebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, input externaldoc.SearchQueryPlanningInput, base externaldoc.WebSearchEvidence) externaldoc.WebSearchEvidence {
	if c == nil || !c.enabled {
		if base.Enabled {
			return cloneReviewWebSearchEvidence(base)
		}
		return externaldoc.WebSearchEvidence{Enabled: false}
	}
	evidence := cloneReviewWebSearchEvidence(base)
	evidence.Enabled = true
	if c.searcher == nil {
		return finalizeReviewWebSearchEvidenceWithError(evidence, "web search evidence searcher is not configured")
	}
	if c.fetcher == nil {
		return finalizeReviewWebSearchEvidenceWithError(evidence, "web search evidence fetcher is not configured")
	}

	selection := selectReviewWebSearchCandidates(externaldoc.BuildSearchQueryCandidates(input), evidence, c.maxQueries)
	if selection.truncated {
		evidence.Truncated = true
	}
	if len(selection.candidates) == 0 {
		return finalizeReviewWebSearchEvidence(evidence)
	}

	var errors []string
	if strings.TrimSpace(evidence.Error) != "" {
		errors = append(errors, evidence.Error)
	}
	docIndex := nextReviewWebSearchExternalDocIndex(evidence.ExternalDocs)
	docIDs := reviewWebSearchExternalDocIDSet(evidence.ExternalDocs)
	for _, candidate := range selection.candidates {
		queryEvidence := externaldoc.WebSearchEvidenceQuery{
			Query:  candidate.Query(),
			Reason: candidate.EvidenceReason(),
		}
		result, err := c.searcher.SearchReviewWeb(ctx, candidate.Query(), c.maxResultsPerQuery)
		if result.Provider != "" && evidence.Provider == "" {
			evidence.Provider = result.Provider
		}
		if err != nil {
			queryEvidence.Error = err.Error()
			errors = append(errors, fmt.Sprintf("%q: %v", candidate.Query(), err))
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
			docID := nextReviewWebSearchExternalDocID(&docIndex, docIDs)
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
	return finalizeReviewWebSearchEvidence(evidence)
}
