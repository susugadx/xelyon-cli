package externaldoc

import (
	"context"
	"time"
)

// SourceCredibility は external_doc の出典信頼度を保守的に分類する。
type SourceCredibility string

const (
	// SourceCredibilityOfficialCandidate は公式 docs らしさを複数 signal で確認できた候補。公式確定ではない。
	SourceCredibilityOfficialCandidate SourceCredibility = "official_candidate"
	// SourceCredibilityThirdParty は非公式・community・tutorial などの signal が明確な出典。
	SourceCredibilityThirdParty SourceCredibility = "third_party"
	// SourceCredibilityUnknown は出典信頼度を公式候補とも third-party とも判定しない状態。
	SourceCredibilityUnknown SourceCredibility = "unknown"
)

// Evidence は検索結果 URL から取得した引用可能 snippet 群を表す。
type Evidence struct {
	DocID                   string            `json:"doc_id"`
	URL                     string            `json:"url"`
	SourceDomain            string            `json:"source_domain,omitempty"`
	SourceCredibility       SourceCredibility `json:"source_credibility,omitempty"`
	SourceCredibilityReason string            `json:"source_credibility_reason,omitempty"`
	FetchedAt               time.Time         `json:"fetched_at,omitempty"`
	StatusCode              int               `json:"status_code,omitempty"`
	ContentType             string            `json:"content_type,omitempty"`
	ContentHash             string            `json:"content_hash,omitempty"`
	Truncated               bool              `json:"truncated,omitempty"`
	Snippets                []SnippetEvidence `json:"snippets,omitempty"`
	Error                   string            `json:"error,omitempty"`
}

// SnippetEvidence は external_doc evidence の引用可能な bounded snippet。
type SnippetEvidence struct {
	SnippetID   string `json:"snippet_id"`
	Content     string `json:"content"`
	ContentHash string `json:"content_hash"`
	Truncated   bool   `json:"truncated,omitempty"`
	FocusTerm   string `json:"focus_term,omitempty"`
	FocusReason string `json:"focus_reason,omitempty"`
}

// WebSearchEvidence は /review 用の外部 Web 検索 evidence を表す。
type WebSearchEvidence struct {
	Enabled      bool                     `json:"enabled"`
	Provider     string                   `json:"provider,omitempty"`
	Queries      []WebSearchEvidenceQuery `json:"queries,omitempty"`
	ExternalDocs []Evidence               `json:"external_docs,omitempty"`
	Error        string                   `json:"error,omitempty"`
	Truncated    bool                     `json:"truncated,omitempty"`
	Inconclusive bool                     `json:"inconclusive,omitempty"`
}

// WebSearchEvidenceQuery は 1 件の検索 query と結果を表す。
type WebSearchEvidenceQuery struct {
	Query   string                    `json:"query"`
	Reason  string                    `json:"reason"`
	Results []WebSearchEvidenceResult `json:"results,omitempty"`
	Error   string                    `json:"error,omitempty"`
}

// WebSearchEvidenceResult は検索結果 URL と discovery-only snippet を表す。
type WebSearchEvidenceResult struct {
	Title        string `json:"title,omitempty"`
	URL          string `json:"url"`
	Snippet      string `json:"snippet,omitempty"`
	SourceDomain string `json:"source_domain,omitempty"`
}

// WebSearchQueryResult は検索 provider と URL 付き結果を表す。
type WebSearchQueryResult struct {
	Provider  string
	Results   []WebSearchEvidenceResult
	Truncated bool
}

// FetchRequest は external_doc fetch 境界へ渡す検索結果 URL と判定 hint。
// URL と DocID は required、FocusTerms は snippet 抽出用の任意 hint、
// SearchResultTitle と QuerySubjectHint は信頼度判定用の任意 hint として扱う。
type FetchRequest struct {
	URL               string
	DocID             string
	FocusTerms        []FocusTerm
	SearchResultTitle string
	QuerySubjectHint  string
}

// FocusTerm は external_doc snippet で優先して引用範囲へ寄せる語句。
type FocusTerm struct {
	Term   string
	Reason string
}

// QueryIntent は external doc 検索 query の意図を表す。
// source credibility や official confirmation の代替ではなく、query planning 用の分類だけを担う。
type QueryIntent string

const (
	// QueryIntentOfficialDocs は公式 docs 候補を探す意図。
	QueryIntentOfficialDocs QueryIntent = "official_docs"
	// QueryIntentAPIDocs は API reference / API docs 候補を探す意図。
	QueryIntentAPIDocs QueryIntent = "api_docs"
	// QueryIntentSecurityAdvisory は security advisory / vulnerability notes 候補を探す意図。
	QueryIntentSecurityAdvisory QueryIntent = "security_advisory"
	// QueryIntentSpec は RFC や protocol specification 候補を探す意図。
	QueryIntentSpec QueryIntent = "spec"
	// QueryIntentFrameworkBehavior は framework / language runtime behavior docs 候補を探す意図。
	QueryIntentFrameworkBehavior QueryIntent = "framework_behavior"
	// QueryIntentFallback は weak input からの低信頼 fallback query 意図。
	QueryIntentFallback QueryIntent = "fallback"
)

// QueryExpectedSourceType は query が期待する外部 source の種類を表す。
type QueryExpectedSourceType string

const (
	// QueryExpectedSourceOfficialDocumentation は公式ドキュメントを期待する。
	QueryExpectedSourceOfficialDocumentation QueryExpectedSourceType = "official_documentation"
	// QueryExpectedSourceAPIReference は API reference を期待する。
	QueryExpectedSourceAPIReference QueryExpectedSourceType = "api_reference"
	// QueryExpectedSourceSecurityAdvisory は security advisory を期待する。
	QueryExpectedSourceSecurityAdvisory QueryExpectedSourceType = "security_advisory"
	// QueryExpectedSourceTechnicalSpecification は RFC / specification を期待する。
	QueryExpectedSourceTechnicalSpecification QueryExpectedSourceType = "technical_specification"
	// QueryExpectedSourceFrameworkDocumentation は framework / language docs を期待する。
	QueryExpectedSourceFrameworkDocumentation QueryExpectedSourceType = "framework_documentation"
	// QueryExpectedSourceGeneralReference は低信頼 fallback の一般 reference を期待する。
	QueryExpectedSourceGeneralReference QueryExpectedSourceType = "general_reference"
)

// QueryConfidence は planner が query を作る根拠の強さを表す。
type QueryConfidence string

const (
	// QueryConfidenceHigh は具体 entity と concrete focus が揃った query。
	QueryConfidenceHigh QueryConfidence = "high"
	// QueryConfidenceMedium は具体 entity はあるが focus がやや広い query。
	QueryConfidenceMedium QueryConfidence = "medium"
	// QueryConfidenceLow は fallback としてのみ使う query。
	QueryConfidenceLow QueryConfidence = "low"
)

// SearchQueryPlanningInput は review evidence から抽出した external doc 検索計画用の入力。
type SearchQueryPlanningInput struct {
	CorpusParts         []string
	GenericImpactTokens []string
	ImpactSurfaces      []SearchQueryPlanImpactSurface
	CandidateRisks      []SearchQueryPlanCandidateRisk
}

// SearchQueryPlanImpactSurface は Pass1 impact surface から externaldoc が受け取る query planning DTO。
type SearchQueryPlanImpactSurface struct {
	ID              string
	Summary         string
	Category        string
	EvidenceSummary string
	Reason          string
}

// SearchQueryPlanCandidateRisk は Pass1 candidate risk から externaldoc が受け取る query planning DTO。
type SearchQueryPlanCandidateRisk struct {
	ID                   string
	Summary              string
	Severity             string
	SurfaceIDs           []string
	EvidenceSummary      string
	VerificationStrategy string
	Status               string
}

// SearchQueryCandidate は fetch に引き継ぐ subject / focus hint 付き検索 query。
// 候補は BuildSearchQueryCandidates が検証済みとして作り、外部 caller は accessor 経由で参照する。
type SearchQueryCandidate struct {
	query              string
	reason             string
	subject            string
	focus              string
	intent             QueryIntent
	expectedSourceType QueryExpectedSourceType
	confidence         QueryConfidence
}

// Fetcher は検索結果 URL から external_doc snippet を取得する境界。
type Fetcher interface {
	FetchExternalDoc(context.Context, FetchRequest) Evidence
}
