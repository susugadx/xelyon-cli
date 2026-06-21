package externaldoc

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
