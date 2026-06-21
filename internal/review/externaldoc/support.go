package externaldoc

import "strings"

// ExternalSupportLevel は external_doc evidence 全体の外部根拠としての強さ。
type ExternalSupportLevel string

const (
	// ExternalSupportLevelNone は外部根拠として使える fetched snippet がない状態。
	ExternalSupportLevelNone ExternalSupportLevel = "none"
	// ExternalSupportLevelWeak は unknown / third-party 中心で公式確認に使えない状態。
	ExternalSupportLevelWeak ExternalSupportLevel = "weak"
	// ExternalSupportLevelPartial は公式候補や snippet が少なく、確認済み仕様とは扱えない状態。
	ExternalSupportLevelPartial ExternalSupportLevel = "partial"
	// ExternalSupportLevelAdequate は複数の一意な公式候補 source と複数の引用可能 snippet が揃った状態。
	ExternalSupportLevelAdequate ExternalSupportLevel = "adequate"
	// ExternalSupportLevelStrong は将来の source ranking / claim relevance 用に予約する。
	ExternalSupportLevelStrong ExternalSupportLevel = "strong"
)

// ExternalSupportSummary は WebSearchEvidence の外部根拠品質を LLM 入力向けに要約する。
type ExternalSupportSummary struct {
	Level                                             ExternalSupportLevel `json:"level"`
	DocCount                                          int                  `json:"doc_count"`
	CitationCapableDocCount                           int                  `json:"citation_capable_doc_count"`
	SnippetCount                                      int                  `json:"snippet_count"`
	CitationCapableSnippetCount                       int                  `json:"citation_capable_snippet_count"`
	OfficialCandidateDocCount                         int                  `json:"official_candidate_doc_count"`
	OfficialCandidateCitationCapableDocCount          int                  `json:"official_candidate_citation_capable_doc_count"`
	OfficialCandidateUniqueCitationCapableSourceCount int                  `json:"official_candidate_unique_citation_capable_source_count"`
	OfficialCandidateCitationCapableSnippetCount      int                  `json:"official_candidate_citation_capable_snippet_count"`
	ThirdPartyDocCount                                int                  `json:"third_party_doc_count"`
	UnknownDocCount                                   int                  `json:"unknown_doc_count"`
	ErrorDocCount                                     int                  `json:"error_doc_count"`
	TruncatedDocCount                                 int                  `json:"truncated_doc_count"`
	TruncatedSnippetCount                             int                  `json:"truncated_snippet_count"`
	OfficialConfirmation                              bool                 `json:"official_confirmation"`
	Warnings                                          []string             `json:"warnings"`
	Reasons                                           []string             `json:"reasons"`
}

// SummarizeExternalSupport は WebSearchEvidence を保守的な external support summary に分類する。
func SummarizeExternalSupport(evidence WebSearchEvidence) ExternalSupportSummary {
	summary := ExternalSupportSummary{
		Level:    ExternalSupportLevelNone,
		Warnings: []string{},
		Reasons:  []string{},
	}

	hasQueryError := false
	hasEvidenceError := strings.TrimSpace(evidence.Error) != ""
	for _, query := range evidence.Queries {
		if strings.TrimSpace(query.Error) != "" {
			hasQueryError = true
			summary.Warnings = appendExternalSupportUnique(summary.Warnings, "web_search_query_error")
		}
	}

	officialCandidateSourceCounter := newExternalSupportUniqueSourceCounter()
	for _, doc := range evidence.ExternalDocs {
		summary.DocCount++
		credibility := normalizeSourceCredibility(doc.SourceCredibility)
		switch credibility {
		case SourceCredibilityOfficialCandidate:
			summary.OfficialCandidateDocCount++
		case SourceCredibilityThirdParty:
			summary.ThirdPartyDocCount++
		default:
			summary.UnknownDocCount++
		}

		if strings.TrimSpace(doc.Error) != "" {
			summary.ErrorDocCount++
			summary.Warnings = appendExternalSupportUnique(summary.Warnings, "external_doc_error")
		}
		if doc.Truncated {
			summary.TruncatedDocCount++
			summary.Warnings = appendExternalSupportUnique(summary.Warnings, "external_doc_truncated")
		}

		citationCapableSnippets := 0
		var citationCapableSnippetHashes []string
		for _, snippet := range doc.Snippets {
			summary.SnippetCount++
			if snippet.Truncated {
				summary.TruncatedSnippetCount++
				summary.Warnings = appendExternalSupportUnique(summary.Warnings, "external_doc_snippet_truncated")
			}
			if externalSupportSnippetCitationCapable(doc, snippet) {
				citationCapableSnippets++
				summary.CitationCapableSnippetCount++
				if credibility == SourceCredibilityOfficialCandidate {
					summary.OfficialCandidateCitationCapableSnippetCount++
					citationCapableSnippetHashes = append(citationCapableSnippetHashes, snippet.ContentHash)
				}
			}
		}
		if citationCapableSnippets > 0 {
			summary.CitationCapableDocCount++
			if credibility == SourceCredibilityOfficialCandidate {
				summary.OfficialCandidateCitationCapableDocCount++
				officialCandidateSourceCounter.add(externalSupportUniqueSourceKeys(doc, citationCapableSnippetHashes))
			}
		}
	}
	summary.OfficialCandidateUniqueCitationCapableSourceCount = officialCandidateSourceCounter.count()

	if summary.UnknownDocCount > 0 {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "unknown_source_credibility")
	}
	if hasEvidenceError {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "web_search_evidence_error")
	}
	if summary.SnippetCount > 0 && summary.CitationCapableSnippetCount == 0 {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "external_docs_have_no_citation_capable_snippets")
	}

	if !evidence.Enabled {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "web_search_evidence_disabled")
		summary.Reasons = append(summary.Reasons, "level=none: web search evidence is disabled")
		return finalizeExternalSupportSummary(summary)
	}
	if evidence.Truncated {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "web_search_evidence_truncated")
	}
	if evidence.Inconclusive {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "web_search_evidence_inconclusive")
	}
	if summary.DocCount == 0 {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "no_external_docs")
		if hasEvidenceError {
			summary.Reasons = append(summary.Reasons, "level=none: web search evidence failed before external docs were fetched")
		} else {
			summary.Reasons = append(summary.Reasons, "level=none: no external docs were fetched")
		}
		return finalizeExternalSupportSummary(summary)
	}
	if summary.CitationCapableSnippetCount == 0 {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "no_citation_capable_snippets")
		if hasEvidenceError {
			summary.Reasons = append(summary.Reasons, "level=none: web search evidence failed before citation-capable external_doc snippets were available")
		} else {
			summary.Reasons = append(summary.Reasons, "level=none: no citation-capable external_doc snippets are available")
		}
		return finalizeExternalSupportSummary(summary)
	}

	if summary.OfficialCandidateUniqueCitationCapableSourceCount == 0 {
		summary.Level = ExternalSupportLevelWeak
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "third_party_or_unknown_only_support")
		summary.Reasons = append(summary.Reasons, "level=weak: citation-capable evidence is only third-party or unknown")
		return finalizeExternalSupportSummary(summary)
	}

	summary.Level = ExternalSupportLevelPartial
	if summary.CitationCapableSnippetCount == 1 {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "single_citation_capable_snippet_support")
		summary.Reasons = append(summary.Reasons, "level=partial: only one citation-capable snippet is available")
	}
	if summary.OfficialCandidateUniqueCitationCapableSourceCount == 1 {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "single_official_candidate_source")
		summary.Reasons = append(summary.Reasons, "level=partial: only one unique citation-capable official_candidate source is available")
	}
	if summary.OfficialCandidateCitationCapableDocCount >= 2 && summary.OfficialCandidateUniqueCitationCapableSourceCount == 1 {
		summary.Warnings = appendExternalSupportUnique(summary.Warnings, "duplicate_official_candidate_source")
		summary.Reasons = append(summary.Reasons, "level=partial: duplicate citation-capable official_candidate docs provide only one unique source")
	}

	if summary.OfficialCandidateUniqueCitationCapableSourceCount >= 2 &&
		summary.CitationCapableSnippetCount >= 2 &&
		!externalSupportHasWeakeningSignals(evidence, summary, hasEvidenceError, hasQueryError) {
		summary.Level = ExternalSupportLevelAdequate
		summary.Reasons = append(summary.Reasons, "level=adequate: multiple unique citation-capable official_candidate sources and snippets are available")
	} else if summary.OfficialCandidateUniqueCitationCapableSourceCount >= 2 && summary.CitationCapableSnippetCount >= 2 {
		summary.Reasons = append(summary.Reasons, "level=partial: truncation, inconclusive, or error signals prevent adequate support")
	}

	return finalizeExternalSupportSummary(summary)
}
