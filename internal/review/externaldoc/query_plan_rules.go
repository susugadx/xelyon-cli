package externaldoc

import "strings"

type searchQueryPlanRule struct {
	match      func(string) bool
	focus      []string
	subject    func(string, string) string
	intent     func(string) QueryIntent
	reasonKind string
}

var searchQueryPlanRules = []searchQueryPlanRule{
	{
		match: func(corpus string) bool {
			return strings.Contains(corpus, "openai") || strings.Contains(corpus, "responses")
		},
		focus: []string{
			searchQueryFocusPreviousResponseID,
			searchQueryFocusResponseFormat,
			searchQueryFocusWebSearch,
			searchQueryFocusToolChoice,
			searchQueryFocusFunctionCalling,
		},
		subject: func(corpus, focus string) string {
			if strings.Contains(corpus, "responses") || focus == searchQueryFocusPreviousResponseID {
				return "OpenAI Responses API"
			}
			return "OpenAI API"
		},
		intent:     fixedSearchQueryPlanIntent(QueryIntentAPIDocs),
		reasonKind: "API contract",
	},
	{
		match: func(corpus string) bool {
			return strings.Contains(corpus, "oauth")
		},
		focus: []string{
			searchQueryFocusRedirectURIText,
			searchQueryFocusRedirectURIField,
			searchQueryFocusAccessToken,
			searchQueryFocusAuthorizationCode,
			searchQueryFocusTokenEndpoint,
		},
		subject:    fixedSearchQueryPlanSubject("OAuth 2.0"),
		intent:     fixedSearchQueryPlanIntent(QueryIntentSpec),
		reasonKind: "protocol/spec",
	},
	{
		match: searchQueryCorpusHasGoFilepathSignal,
		focus: []string{
			searchQueryFocusPathTraversal,
			searchQueryFocusDirectoryTraversal,
			searchQueryFocusSymlink,
			searchQueryFocusEvalSymlink,
			searchQueryFocusFilepathEvalSymlink,
			searchQueryFocusFilepathClean,
		},
		subject: fixedSearchQueryPlanSubject("Go filepath package"),
		intent: func(focus string) QueryIntent {
			if searchQuerySecuritySignal(focus) {
				return QueryIntentSecurityAdvisory
			}
			return QueryIntentFrameworkBehavior
		},
		reasonKind: "path/security",
	},
	{
		match: func(corpus string) bool {
			return strings.Contains(corpus, "model context protocol") || strings.Contains(corpus, " mcp ")
		},
		focus: []string{
			searchQueryFocusToolCalls,
			searchQueryFocusToolChoice,
			searchQueryFocusJSONSchema,
		},
		subject:    fixedSearchQueryPlanSubject("Model Context Protocol"),
		intent:     fixedSearchQueryPlanIntent(QueryIntentSpec),
		reasonKind: "protocol/spec",
	},
}

func fixedSearchQueryPlanSubject(subject string) func(string, string) string {
	return func(string, string) string {
		return subject
	}
}

func fixedSearchQueryPlanIntent(intent QueryIntent) func(string) QueryIntent {
	return func(string) QueryIntent {
		return intent
	}
}

func buildPlanSearchQueryCandidates(input SearchQueryPlanningInput, corpus string) []SearchQueryCandidate {
	planCorpus := searchQueryPlanCorpus(input)
	if planCorpus == "" {
		return nil
	}

	var candidates []SearchQueryCandidate
	for _, rule := range searchQueryPlanRules {
		candidates = append(candidates, rule.buildCandidates(planCorpus)...)
	}

	if len(candidates) == 0 && len(input.ImpactSurfaces)+len(input.CandidateRisks) > 0 {
		for _, subject := range SearchSubjectsForCorpus(corpus) {
			for _, focus := range searchQueryFocusTokens(input.GenericImpactTokens, corpus) {
				candidate, ok := newSearchQueryCandidate(searchQueryCandidateSpec{
					Subject:            subject,
					Focus:              focus,
					Query:              buildSearchQueryText(subject, focus, QueryIntentOfficialDocs),
					Reason:             "pass1 plan external contract signal: " + subject + " / " + focus,
					Intent:             QueryIntentOfficialDocs,
					ExpectedSourceType: QueryExpectedSourceOfficialDocumentation,
					Confidence:         QueryConfidenceMedium,
				})
				if ok {
					candidates = append(candidates, candidate)
				}
			}
		}
	}
	return candidates
}

func (r searchQueryPlanRule) buildCandidates(planCorpus string) []SearchQueryCandidate {
	if r.match == nil || !r.match(planCorpus) {
		return nil
	}
	focuses := presentSearchQueryFocuses(planCorpus, r.focus)
	if len(focuses) == 0 || r.subject == nil || r.intent == nil || r.reasonKind == "" {
		return nil
	}
	candidates := make([]SearchQueryCandidate, 0, len(focuses))
	for _, focus := range focuses {
		subject := r.subject(planCorpus, focus)
		intent := r.intent(focus)
		candidate, ok := newSearchQueryCandidate(searchQueryCandidateSpec{
			Subject:            subject,
			Focus:              focus,
			Query:              buildSearchQueryText(subject, focus, intent),
			Reason:             "pass1 plan " + r.reasonKind + " signal: " + subject + " / " + focus,
			Intent:             intent,
			ExpectedSourceType: expectedSourceTypeForIntent(intent),
			Confidence:         QueryConfidenceHigh,
		})
		if ok {
			candidates = append(candidates, candidate)
		}
	}
	return candidates
}

func searchQueryPlanCorpus(input SearchQueryPlanningInput) string {
	if len(input.ImpactSurfaces) == 0 && len(input.CandidateRisks) == 0 {
		return ""
	}
	return " " + strings.ToLower(strings.Join(searchQueryPlanCorpusParts(input), "\n")) + " "
}

func searchQueryPlanCorpusParts(input SearchQueryPlanningInput) []string {
	parts := make([]string, 0, len(input.ImpactSurfaces)*5+len(input.CandidateRisks)*7)
	for _, surface := range input.ImpactSurfaces {
		parts = append(parts, surface.ID, surface.Summary, surface.Category, surface.EvidenceSummary, surface.Reason)
	}
	for _, risk := range input.CandidateRisks {
		parts = append(parts, risk.ID, risk.Summary, risk.Severity, risk.EvidenceSummary, risk.VerificationStrategy, risk.Status)
		parts = append(parts, risk.SurfaceIDs...)
	}
	return parts
}

func presentSearchQueryFocuses(corpus string, candidates []string) []string {
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.Contains(corpus, strings.ToLower(candidate)) {
			result = append(result, candidate)
		}
	}
	return result
}
