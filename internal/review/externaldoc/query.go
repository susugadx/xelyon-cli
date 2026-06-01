package externaldoc

import "strings"

const defaultWebSearchQueryCandidateCap = 9

const (
	searchQueryFocusPreviousResponseID  = "previous_response_id"
	searchQueryFocusResponseFormat      = "response_format"
	searchQueryFocusWebSearch           = "web_search"
	searchQueryFocusResponsesAPI        = "responses API"
	searchQueryFocusFunctionCalling     = "function calling"
	searchQueryFocusToolChoice          = "tool_choice"
	searchQueryFocusToolCalls           = "tool calls"
	searchQueryFocusServiceTier         = "service_tier"
	searchQueryFocusAnthropicVersion    = "anthropic_version"
	searchQueryFocusCacheControl        = "cache_control"
	searchQueryFocusEventStream         = "text/event-stream"
	searchQueryFocusJSONSchema          = "JSON schema"
	searchQueryFocusRedirectURIField    = "redirect_uri"
	searchQueryFocusRedirectURIText     = "redirect URI"
	searchQueryFocusAccessToken         = "access token"
	searchQueryFocusAuthorizationCode   = "authorization code"
	searchQueryFocusOAuth20             = "OAuth 2.0"
	searchQueryFocusFilepathEvalSymlink = "filepath.EvalSymlinks"
	searchQueryFocusEvalSymlink         = "EvalSymlinks"
	searchQueryFocusFilepathClean       = "filepath.Clean"
	searchQueryFocusPathTraversal       = "path traversal"
	searchQueryFocusDirectoryTraversal  = "directory traversal"
	searchQueryFocusSymlink             = "symlink"
	searchQueryFocusTokenEndpoint       = "token endpoint"
)

type searchQueryFocusTokenCatalogEntry struct {
	token     string
	preferred bool
}

var searchQueryFocusTokenCatalog = []searchQueryFocusTokenCatalogEntry{
	{token: searchQueryFocusPreviousResponseID, preferred: true},
	{token: searchQueryFocusResponseFormat, preferred: true},
	{token: searchQueryFocusWebSearch, preferred: true},
	{token: searchQueryFocusResponsesAPI, preferred: true},
	{token: searchQueryFocusFunctionCalling, preferred: true},
	{token: searchQueryFocusToolChoice, preferred: true},
	{token: searchQueryFocusToolCalls, preferred: true},
	{token: searchQueryFocusServiceTier, preferred: true},
	{token: searchQueryFocusAnthropicVersion, preferred: true},
	{token: searchQueryFocusCacheControl, preferred: true},
	{token: searchQueryFocusEventStream, preferred: true},
	{token: searchQueryFocusJSONSchema, preferred: true},
	{token: searchQueryFocusRedirectURIField, preferred: true},
	{token: searchQueryFocusRedirectURIText, preferred: true},
	{token: searchQueryFocusAccessToken, preferred: true},
	{token: searchQueryFocusAuthorizationCode, preferred: true},
	{token: searchQueryFocusOAuth20, preferred: true},
	{token: searchQueryFocusFilepathEvalSymlink, preferred: true},
	{token: searchQueryFocusEvalSymlink, preferred: true},
	{token: searchQueryFocusFilepathClean, preferred: true},
	{token: searchQueryFocusPathTraversal, preferred: true},
	{token: searchQueryFocusDirectoryTraversal, preferred: true},
	{token: searchQueryFocusSymlink, preferred: true},
	{token: searchQueryFocusTokenEndpoint},
}

var searchQueryAllowedFocusTokenKeys = buildSearchQueryFocusTokenKeySet(searchQueryFocusTokenCatalog)

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

// BuildFetchRequest は検索候補と検索結果を external doc fetch request に写像する。
func BuildFetchRequest(candidate SearchQueryCandidate, result WebSearchEvidenceResult, genericTokens []string, docID string) FetchRequest {
	return FetchRequest{
		URL:               result.URL,
		DocID:             docID,
		FocusTerms:        BuildFocusTerms(candidate.query, candidate.subject, candidate.focus, result.Title, result.Snippet, genericTokens),
		SearchResultTitle: result.Title,
		QuerySubjectHint:  candidate.subject,
	}
}

func searchQueryPlanningCorpus(input SearchQueryPlanningInput) string {
	parts := make([]string, 0, len(input.CorpusParts)+len(input.GenericImpactTokens)+len(input.ImpactSurfaces)*5+len(input.CandidateRisks)*7)
	parts = append(parts, input.CorpusParts...)
	parts = append(parts, input.GenericImpactTokens...)
	parts = append(parts, searchQueryPlanCorpusParts(input)...)
	return strings.ToLower(strings.Join(parts, "\n"))
}

func searchQueryFocusTokens(genericTokens []string, corpus string) []string {
	var result []string
	seen := make(map[string]struct{})
	for _, entry := range searchQueryFocusTokenCatalog {
		if !entry.preferred {
			continue
		}
		token := entry.token
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

func buildSearchQueryText(subject, focus string, intent QueryIntent) string {
	subject = strings.TrimSpace(subject)
	focus = strings.TrimSpace(focus)
	switch intent {
	case QueryIntentSpec:
		return subject + " " + focus + " specification"
	case QueryIntentSecurityAdvisory:
		return subject + " " + focus + " security advisory"
	case QueryIntentFrameworkBehavior:
		return subject + " " + focus + " documentation"
	case QueryIntentFallback:
		return subject + " " + focus + " reference"
	default:
		return subject + " " + focus + " official documentation"
	}
}

func classifySearchQueryIntent(subject, focus, corpus string) QueryIntent {
	lowerSubject := strings.ToLower(subject)
	lowerFocus := strings.ToLower(focus)
	switch {
	case strings.Contains(lowerSubject, "oauth") || strings.Contains(lowerSubject, "model context protocol"):
		return QueryIntentSpec
	case searchQuerySecuritySignal(lowerFocus):
		return QueryIntentSecurityAdvisory
	case strings.Contains(lowerSubject, "filepath") || strings.Contains(lowerSubject, "cloudflare workers"):
		return QueryIntentFrameworkBehavior
	case strings.Contains(lowerSubject, "api"):
		return QueryIntentAPIDocs
	case strings.Contains(corpus, "official") || strings.Contains(corpus, "docs"):
		return QueryIntentOfficialDocs
	default:
		return QueryIntentFallback
	}
}

func expectedSourceTypeForIntent(intent QueryIntent) QueryExpectedSourceType {
	switch intent {
	case QueryIntentAPIDocs:
		return QueryExpectedSourceAPIReference
	case QueryIntentSecurityAdvisory:
		return QueryExpectedSourceSecurityAdvisory
	case QueryIntentSpec:
		return QueryExpectedSourceTechnicalSpecification
	case QueryIntentFrameworkBehavior:
		return QueryExpectedSourceFrameworkDocumentation
	case QueryIntentFallback:
		return QueryExpectedSourceGeneralReference
	default:
		return QueryExpectedSourceOfficialDocumentation
	}
}

func searchQueryConfidenceForSubjectFocus(subject, focus string) QueryConfidence {
	if searchQueryGenericFocusTokenIsConcrete(focus) || searchQueryPreferredFocusTokenIsAllowed(focus) {
		if classifySearchQueryIntent(subject, focus, "") == QueryIntentFallback {
			return QueryConfidenceLow
		}
		return QueryConfidenceHigh
	}
	return QueryConfidenceLow
}

func searchQueryCorpusHasGoFilepathSignal(corpus string) bool {
	return strings.Contains(corpus, "path/filepath") ||
		strings.Contains(corpus, "filepath.") ||
		strings.Contains(corpus, " filepath ") ||
		strings.Contains(corpus, "evalsymlinks")
}

func searchQuerySecuritySignal(value string) bool {
	lower := strings.ToLower(strings.Join(strings.Fields(value), " "))
	if lower == "symlink" {
		return true
	}
	for _, signal := range []string{"security", "advisory", "vulnerability", "cve", "traversal"} {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

func searchQueryPreferredFocusTokenIsAllowed(token string) bool {
	_, ok := searchQueryAllowedFocusTokenKeys[searchQueryFocusTokenKey(token)]
	return ok
}

func buildSearchQueryFocusTokenKeySet(entries []searchQueryFocusTokenCatalogEntry) map[string]struct{} {
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		key := searchQueryFocusTokenKey(entry.token)
		if key != "" {
			seen[key] = struct{}{}
		}
	}
	return seen
}

func searchQueryFocusTokenKey(token string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(token)), " "))
}

func normalizeSearchQueryText(query string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(query)), " ")
}

func normalizeSearchQueryDedupeKey(query string) string {
	return strings.ToLower(strings.ReplaceAll(normalizeSearchQueryText(query), "_", " "))
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
