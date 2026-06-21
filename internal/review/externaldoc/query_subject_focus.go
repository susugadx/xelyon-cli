package externaldoc

import "strings"

func buildSubjectFocusSearchQueryCandidates(input SearchQueryPlanningInput, corpus string) []SearchQueryCandidate {
	externalSubjects := SearchSubjectsForCorpus(corpus)
	if len(externalSubjects) == 0 {
		return nil
	}
	focusTokens := searchQueryFocusTokens(input.GenericImpactTokens, corpus)
	if len(focusTokens) == 0 {
		return nil
	}

	candidates := make([]SearchQueryCandidate, 0, len(externalSubjects)*len(focusTokens))
	for _, subject := range externalSubjects {
		for _, focus := range focusTokens {
			intent := classifySearchQueryIntent(subject, focus, corpus)
			candidate, ok := newSearchQueryCandidate(searchQueryCandidateSpec{
				Subject:            subject,
				Focus:              focus,
				Query:              buildSearchQueryText(subject, focus, intent),
				Reason:             "changed external contract token: " + subject + " / " + focus,
				Intent:             intent,
				ExpectedSourceType: expectedSourceTypeForIntent(intent),
				Confidence:         searchQueryConfidenceForSubjectFocus(subject, focus),
			})
			if !ok {
				continue
			}
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func searchQueryPlanningCorpus(input SearchQueryPlanningInput) string {
	parts := make([]string, 0, len(input.CorpusParts)+len(input.GenericImpactTokens)+len(input.ImpactSurfaces)*5+len(input.CandidateRisks)*7)
	parts = append(parts, input.CorpusParts...)
	parts = append(parts, input.GenericImpactTokens...)
	parts = append(parts, searchQueryPlanCorpusParts(input)...)
	return strings.ToLower(strings.Join(parts, "\n"))
}
