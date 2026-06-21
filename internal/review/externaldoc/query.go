package externaldoc

import "strings"

const defaultWebSearchQueryCandidateCap = 9

// BuildSearchQueryCandidates は external doc 検索用の focused query 候補を作る。
func BuildSearchQueryCandidates(input SearchQueryPlanningInput) []SearchQueryCandidate {
	corpus := searchQueryPlanningCorpus(input)
	candidates := make([]SearchQueryCandidate, 0, defaultWebSearchQueryCandidateCap)
	seen := make(map[string]struct{})

	appendCandidate := func(candidate SearchQueryCandidate) bool {
		if !candidate.valid() {
			return false
		}
		key := normalizeSearchQueryDedupeKey(candidate.query)
		if key == "" {
			return false
		}
		if _, exists := seen[key]; exists {
			return false
		}
		seen[key] = struct{}{}
		candidates = append(candidates, candidate)
		return len(candidates) >= defaultWebSearchQueryCandidateCap
	}

	for _, candidate := range buildPlanSearchQueryCandidates(input, corpus) {
		if appendCandidate(candidate) {
			return candidates
		}
	}
	for _, candidate := range buildSubjectFocusSearchQueryCandidates(input, corpus) {
		if appendCandidate(candidate) {
			return candidates
		}
	}
	return candidates
}

// Query は provider に渡す検索 query を返す。
func (c SearchQueryCandidate) Query() string {
	return c.query
}

// Reason は query planning の根拠を返す。
func (c SearchQueryCandidate) Reason() string {
	return c.reason
}

// EvidenceReason は WebSearchEvidenceQuery.Reason に載せる互換的な metadata 付き根拠を返す。
func (c SearchQueryCandidate) EvidenceReason() string {
	if !c.valid() {
		return ""
	}
	parts := []string{
		"intent=" + string(c.intent),
		"expected_source_type=" + string(c.expectedSourceType),
		"confidence=" + string(c.confidence),
	}
	if c.reason != "" {
		parts = append(parts, "reason="+c.reason)
	}
	return strings.Join(parts, "; ")
}

// Subject は query の主対象を返す。
func (c SearchQueryCandidate) Subject() string {
	return c.subject
}

// Focus は query の具体 focus を返す。
func (c SearchQueryCandidate) Focus() string {
	return c.focus
}

// Intent は query planning 上の意図を返す。
func (c SearchQueryCandidate) Intent() QueryIntent {
	return c.intent
}

// ExpectedSourceType は query が期待する source 種別を返す。
func (c SearchQueryCandidate) ExpectedSourceType() QueryExpectedSourceType {
	return c.expectedSourceType
}

// Confidence は planner が query を作る根拠の強さを返す。
func (c SearchQueryCandidate) Confidence() QueryConfidence {
	return c.confidence
}

// SearchQueryDedupeKey は query dedupe 用の正規化キーを返す。
func SearchQueryDedupeKey(query string) string {
	return normalizeSearchQueryDedupeKey(query)
}

type searchQueryCandidateSpec struct {
	Query              string
	Reason             string
	Subject            string
	Focus              string
	Intent             QueryIntent
	ExpectedSourceType QueryExpectedSourceType
	Confidence         QueryConfidence
}

func newSearchQueryCandidate(spec searchQueryCandidateSpec) (SearchQueryCandidate, bool) {
	query := normalizeSearchQueryText(spec.Query)
	subject := strings.TrimSpace(spec.Subject)
	focus := strings.TrimSpace(spec.Focus)
	reason := strings.Join(strings.Fields(strings.TrimSpace(spec.Reason)), " ")
	if query == "" || subject == "" || focus == "" || reason == "" {
		return SearchQueryCandidate{}, false
	}
	if normalizeSearchQueryDedupeKey(subject) == normalizeSearchQueryDedupeKey(focus) {
		return SearchQueryCandidate{}, false
	}
	if !spec.Intent.valid() || !spec.ExpectedSourceType.valid() || !spec.Confidence.valid() {
		return SearchQueryCandidate{}, false
	}
	if !searchQueryGenericFocusTokenIsConcrete(focus) && !searchQueryPreferredFocusTokenIsAllowed(focus) {
		return SearchQueryCandidate{}, false
	}
	return SearchQueryCandidate{
		query:              query,
		reason:             reason,
		subject:            subject,
		focus:              focus,
		intent:             spec.Intent,
		expectedSourceType: spec.ExpectedSourceType,
		confidence:         spec.Confidence,
	}, true
}

func (c SearchQueryCandidate) valid() bool {
	return normalizeSearchQueryText(c.query) != "" &&
		strings.TrimSpace(c.reason) != "" &&
		strings.TrimSpace(c.subject) != "" &&
		strings.TrimSpace(c.focus) != "" &&
		c.intent.valid() &&
		c.expectedSourceType.valid() &&
		c.confidence.valid()
}

func (i QueryIntent) valid() bool {
	switch i {
	case QueryIntentOfficialDocs,
		QueryIntentAPIDocs,
		QueryIntentSecurityAdvisory,
		QueryIntentSpec,
		QueryIntentFrameworkBehavior,
		QueryIntentFallback:
		return true
	default:
		return false
	}
}

func (t QueryExpectedSourceType) valid() bool {
	switch t {
	case QueryExpectedSourceOfficialDocumentation,
		QueryExpectedSourceAPIReference,
		QueryExpectedSourceSecurityAdvisory,
		QueryExpectedSourceTechnicalSpecification,
		QueryExpectedSourceFrameworkDocumentation,
		QueryExpectedSourceGeneralReference:
		return true
	default:
		return false
	}
}

func (c QueryConfidence) valid() bool {
	switch c {
	case QueryConfidenceHigh, QueryConfidenceMedium, QueryConfidenceLow:
		return true
	default:
		return false
	}
}
