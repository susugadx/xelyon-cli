package review

import "github.com/susugadx/xelyon-cli/internal/review/externaldoc"

// ReviewWebSearchEvidence は /review 用の外部 Web 検索 evidence を表す。
type ReviewWebSearchEvidence = externaldoc.WebSearchEvidence

// ReviewWebSearchEvidenceQuery は 1 件の検索 query と結果を表す。
type ReviewWebSearchEvidenceQuery = externaldoc.WebSearchEvidenceQuery

// ReviewWebSearchEvidenceResult は検索結果 URL と discovery-only snippet を表す。
type ReviewWebSearchEvidenceResult = externaldoc.WebSearchEvidenceResult

// ReviewWebSearchQueryResult は検索 provider と URL 付き結果を表す。
type ReviewWebSearchQueryResult = externaldoc.WebSearchQueryResult
