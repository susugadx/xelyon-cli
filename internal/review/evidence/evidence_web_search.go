package evidence

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
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
	return c.collectWebSearchEvidence(ctx, bundle, buildReviewWebSearchQueryPlanningInput(bundle), ReviewWebSearchEvidence{})
}

// CollectPostPass1WebSearchEvidence は Pass1 probe plan から追加の外部仕様 evidence を収集し、既存 evidence に merge する。
func (c *ReviewWebSearchEvidenceCollector) CollectPostPass1WebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, plan reviewprobe.ReviewProbePlan) ReviewWebSearchEvidence {
	return c.collectWebSearchEvidence(ctx, bundle, buildReviewPostPass1WebSearchQueryPlanningInput(plan), bundle.WebSearchEvidence)
}

func (c *ReviewWebSearchEvidenceCollector) collectWebSearchEvidence(ctx context.Context, bundle ReviewEvidenceBundle, input externaldoc.SearchQueryPlanningInput, base ReviewWebSearchEvidence) ReviewWebSearchEvidence {
	if c == nil || !c.enabled {
		if base.Enabled {
			return cloneReviewWebSearchEvidence(base)
		}
		return ReviewWebSearchEvidence{Enabled: false}
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
		queryEvidence := ReviewWebSearchEvidenceQuery{
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

type reviewWebSearchCandidateSelection struct {
	candidates []externaldoc.SearchQueryCandidate
	truncated  bool
}

func selectReviewWebSearchCandidates(candidates []externaldoc.SearchQueryCandidate, evidence ReviewWebSearchEvidence, maxQueries int) reviewWebSearchCandidateSelection {
	candidates = dedupeReviewWebSearchCandidatesAgainstEvidence(candidates, evidence)
	if len(candidates) == 0 {
		return reviewWebSearchCandidateSelection{}
	}
	remainingQueries := maxQueries - len(evidence.Queries)
	if remainingQueries <= 0 {
		return reviewWebSearchCandidateSelection{truncated: true}
	}
	if len(candidates) > remainingQueries {
		return reviewWebSearchCandidateSelection{
			candidates: candidates[:remainingQueries],
			truncated:  true,
		}
	}
	return reviewWebSearchCandidateSelection{candidates: candidates}
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

func buildReviewPostPass1WebSearchQueryPlanningInput(plan reviewprobe.ReviewProbePlan) externaldoc.SearchQueryPlanningInput {
	surfaces := make([]externaldoc.SearchQueryPlanImpactSurface, 0, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		surfaces = append(surfaces, externaldoc.SearchQueryPlanImpactSurface{
			ID:              surface.ID,
			Summary:         surface.Summary,
			Category:        string(surface.Category),
			EvidenceSummary: surface.EvidenceSummary,
			Reason:          surface.Reason,
		})
	}
	risks := make([]externaldoc.SearchQueryPlanCandidateRisk, 0, len(plan.CandidateRisks))
	for _, risk := range plan.CandidateRisks {
		risks = append(risks, externaldoc.SearchQueryPlanCandidateRisk{
			ID:                   risk.ID,
			Summary:              risk.Summary,
			Severity:             string(risk.Severity),
			SurfaceIDs:           append([]string(nil), risk.SurfaceIDs...),
			EvidenceSummary:      risk.EvidenceSummary,
			VerificationStrategy: risk.VerificationStrategy,
			Status:               string(risk.Status),
		})
	}
	return externaldoc.SearchQueryPlanningInput{
		ImpactSurfaces: surfaces,
		CandidateRisks: risks,
	}
}

func dedupeReviewWebSearchCandidatesAgainstEvidence(candidates []externaldoc.SearchQueryCandidate, evidence ReviewWebSearchEvidence) []externaldoc.SearchQueryCandidate {
	if len(candidates) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(evidence.Queries)+len(candidates))
	for _, query := range evidence.Queries {
		key := externaldoc.SearchQueryDedupeKey(query.Query)
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	result := make([]externaldoc.SearchQueryCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		key := externaldoc.SearchQueryDedupeKey(candidate.Query())
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result
}

func finalizeReviewWebSearchEvidenceWithError(evidence ReviewWebSearchEvidence, message string) ReviewWebSearchEvidence {
	evidence.Error = appendReviewWebSearchEvidenceError(evidence.Error, message)
	return finalizeReviewWebSearchEvidence(evidence)
}

func finalizeReviewWebSearchEvidence(evidence ReviewWebSearchEvidence) ReviewWebSearchEvidence {
	evidence.Inconclusive = !externaldoc.HasFetchedSnippet(evidence.ExternalDocs)
	return evidence
}

func appendReviewWebSearchEvidenceError(existing, message string) string {
	existing = strings.TrimSpace(existing)
	message = strings.TrimSpace(message)
	switch {
	case existing == "":
		return message
	case message == "":
		return existing
	default:
		return existing + "; " + message
	}
}

func cloneReviewWebSearchEvidence(evidence ReviewWebSearchEvidence) ReviewWebSearchEvidence {
	clone := evidence
	clone.Queries = append([]ReviewWebSearchEvidenceQuery(nil), evidence.Queries...)
	for i := range clone.Queries {
		clone.Queries[i].Results = append([]ReviewWebSearchEvidenceResult(nil), evidence.Queries[i].Results...)
	}
	clone.ExternalDocs = append([]ReviewExternalDocEvidence(nil), evidence.ExternalDocs...)
	for i := range clone.ExternalDocs {
		clone.ExternalDocs[i].Snippets = append([]ReviewExternalDocSnippetEvidence(nil), evidence.ExternalDocs[i].Snippets...)
	}
	return clone
}

var reviewWebSearchExternalDocIDRE = regexp.MustCompile(`^external-doc-(\d+)$`)

func nextReviewWebSearchExternalDocIndex(docs []ReviewExternalDocEvidence) int {
	next := len(docs) + 1
	for _, doc := range docs {
		matches := reviewWebSearchExternalDocIDRE.FindStringSubmatch(doc.DocID)
		if len(matches) != 2 {
			continue
		}
		index, err := strconv.Atoi(matches[1])
		if err == nil && index >= next {
			next = index + 1
		}
	}
	return next
}

func reviewWebSearchExternalDocIDSet(docs []ReviewExternalDocEvidence) map[string]struct{} {
	seen := make(map[string]struct{}, len(docs))
	for _, doc := range docs {
		if doc.DocID != "" {
			seen[doc.DocID] = struct{}{}
		}
	}
	return seen
}

func nextReviewWebSearchExternalDocID(nextIndex *int, seen map[string]struct{}) string {
	if *nextIndex <= 0 {
		*nextIndex = 1
	}
	for {
		docID := fmt.Sprintf("external-doc-%d", *nextIndex)
		if _, exists := seen[docID]; !exists {
			seen[docID] = struct{}{}
			return docID
		}
		*nextIndex++
	}
}
