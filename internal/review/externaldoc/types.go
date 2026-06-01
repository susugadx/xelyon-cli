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

// SearchQueryPlanningInput は review evidence から抽出した external doc 検索計画用の入力。
type SearchQueryPlanningInput struct {
	CorpusParts         []string
	GenericImpactTokens []string
}

// SearchQueryCandidate は fetch に引き継ぐ subject / focus hint 付き検索 query。
type SearchQueryCandidate struct {
	Query   string
	Reason  string
	Subject string
	Focus   string
}

// Fetcher は検索結果 URL から external_doc snippet を取得する境界。
type Fetcher interface {
	FetchExternalDoc(context.Context, FetchRequest) Evidence
}
