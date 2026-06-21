package evidence

import "github.com/susugadx/xelyon-cli/internal/review/externaldoc"

type reviewWebSearchCandidateSelection struct {
	candidates []externaldoc.SearchQueryCandidate
	truncated  bool
}

func selectReviewWebSearchCandidates(candidates []externaldoc.SearchQueryCandidate, evidence externaldoc.WebSearchEvidence, maxQueries int) reviewWebSearchCandidateSelection {
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

func limitReviewWebSearchEvidenceResults(results []externaldoc.WebSearchEvidenceResult, maxResults int) ([]externaldoc.WebSearchEvidenceResult, bool) {
	truncated := false
	if maxResults > 0 && len(results) > maxResults {
		results = results[:maxResults]
		truncated = true
	}
	return append([]externaldoc.WebSearchEvidenceResult(nil), results...), truncated
}

func dedupeReviewWebSearchCandidatesAgainstEvidence(candidates []externaldoc.SearchQueryCandidate, evidence externaldoc.WebSearchEvidence) []externaldoc.SearchQueryCandidate {
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
