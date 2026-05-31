package review

import (
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

// ReviewExternalDocSourceCredibility は external_doc の出典信頼度を保守的に分類する。
type ReviewExternalDocSourceCredibility = externaldoc.SourceCredibility

const (
	// ReviewExternalDocSourceCredibilityOfficialCandidate は公式 docs らしさを複数 signal で確認できた候補。
	ReviewExternalDocSourceCredibilityOfficialCandidate = externaldoc.SourceCredibilityOfficialCandidate
	// ReviewExternalDocSourceCredibilityThirdParty は非公式・community・tutorial などの signal が明確な出典。
	ReviewExternalDocSourceCredibilityThirdParty = externaldoc.SourceCredibilityThirdParty
	// ReviewExternalDocSourceCredibilityUnknown は出典信頼度を公式候補とも third-party とも判定しない状態。
	ReviewExternalDocSourceCredibilityUnknown = externaldoc.SourceCredibilityUnknown
)

// ReviewExternalDocEvidence は検索結果 URL から取得した引用可能 snippet 群を表す。
type ReviewExternalDocEvidence = externaldoc.Evidence

// ReviewExternalDocSnippetEvidence は external_doc evidence の引用可能な bounded snippet。
type ReviewExternalDocSnippetEvidence = externaldoc.SnippetEvidence

// ReviewExternalDocFetchRequest は external_doc fetch 境界へ渡す検索結果 URL と判定 hint。
type ReviewExternalDocFetchRequest = externaldoc.FetchRequest

// ReviewExternalDocFocusTerm は external_doc snippet で優先して引用範囲へ寄せる語句。
type ReviewExternalDocFocusTerm = externaldoc.FocusTerm

// ReviewExternalDocFetcher は検索結果 URL から external_doc snippet を取得する境界。
type ReviewExternalDocFetcher = externaldoc.Fetcher

// HTTPReviewExternalDocFetcher は HTTPS URL から bounded text snippet を取得する。
type HTTPReviewExternalDocFetcher = externaldoc.HTTPFetcher

// NewHTTPReviewExternalDocFetcher は external doc fetcher を構築する。
func NewHTTPReviewExternalDocFetcher(client *http.Client) *HTTPReviewExternalDocFetcher {
	return externaldoc.NewHTTPFetcher(client)
}
