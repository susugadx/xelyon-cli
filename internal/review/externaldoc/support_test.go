package externaldoc

import (
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestSummarizeExternalSupportNoDocsIsNone(t *testing.T) {
	summary := SummarizeExternalSupport(WebSearchEvidence{Enabled: true})

	if summary.Level != ExternalSupportLevelNone {
		t.Fatalf("Level = %q, want none", summary.Level)
	}
	if summary.OfficialConfirmation {
		t.Fatal("OfficialConfirmation = true, want false")
	}
	if !slices.Contains(summary.Warnings, "no_external_docs") {
		t.Fatalf("Warnings = %#v, want no_external_docs", summary.Warnings)
	}
}

func TestSummarizeExternalSupportDisabledAndFailedWithoutCitationAreNone(t *testing.T) {
	tests := []struct {
		name     string
		evidence WebSearchEvidence
		warning  string
	}{
		{
			name: "disabled",
			evidence: WebSearchEvidence{
				Enabled:      false,
				ExternalDocs: []Evidence{newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "snippet")},
			},
		},
		{
			name: "failed",
			evidence: WebSearchEvidence{
				Enabled: true,
				Error:   "search failed",
			},
			warning: "web_search_evidence_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := SummarizeExternalSupport(tt.evidence)

			if summary.Level != ExternalSupportLevelNone {
				t.Fatalf("Level = %q, want none", summary.Level)
			}
			if summary.OfficialConfirmation {
				t.Fatal("OfficialConfirmation = true, want false")
			}
			if tt.warning != "" && !slices.Contains(summary.Warnings, tt.warning) {
				t.Fatalf("Warnings = %#v, want %s", summary.Warnings, tt.warning)
			}
		})
	}
}

func TestSummarizeExternalSupportEvidenceErrorWithFetchedOfficialDocsIsPartial(t *testing.T) {
	summary := SummarizeExternalSupport(WebSearchEvidence{
		Enabled: true,
		Error:   "search failed after partial fetch",
		ExternalDocs: []Evidence{
			newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "first official snippet"),
			newExternalSupportDocForTest("doc-2", SourceCredibilityOfficialCandidate, false, "second official snippet"),
		},
	})

	if summary.Level != ExternalSupportLevelPartial {
		t.Fatalf("Level = %q, want partial", summary.Level)
	}
	if summary.OfficialConfirmation {
		t.Fatal("OfficialConfirmation = true, want false")
	}
	if summary.OfficialCandidateUniqueCitationCapableSourceCount != 2 {
		t.Fatalf("OfficialCandidateUniqueCitationCapableSourceCount = %d, want 2", summary.OfficialCandidateUniqueCitationCapableSourceCount)
	}
	for _, want := range []string{"web_search_evidence_error"} {
		if !slices.Contains(summary.Warnings, want) {
			t.Fatalf("Warnings = %#v, want %s", summary.Warnings, want)
		}
	}
	if !slices.Contains(summary.Reasons, "level=partial: truncation, inconclusive, or error signals prevent adequate support") {
		t.Fatalf("Reasons = %#v, want partial weakening reason", summary.Reasons)
	}
}

func TestSummarizeExternalSupportEvidenceErrorWithoutCitationCapableSnippetsIsNone(t *testing.T) {
	summary := SummarizeExternalSupport(WebSearchEvidence{
		Enabled: true,
		Error:   "search failed before fetch",
		ExternalDocs: []Evidence{
			{
				DocID:             "doc-1",
				URL:               "https://docs.example.test/doc-1",
				SourceCredibility: SourceCredibilityOfficialCandidate,
				FetchedAt:         time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
				Snippets: []SnippetEvidence{
					{SnippetID: "doc-1-snippet-1", Content: "missing hash"},
				},
			},
		},
	})

	if summary.Level != ExternalSupportLevelNone {
		t.Fatalf("Level = %q, want none", summary.Level)
	}
	if summary.OfficialConfirmation {
		t.Fatal("OfficialConfirmation = true, want false")
	}
	for _, want := range []string{"web_search_evidence_error", "no_citation_capable_snippets"} {
		if !slices.Contains(summary.Warnings, want) {
			t.Fatalf("Warnings = %#v, want %s", summary.Warnings, want)
		}
	}
}

func TestSummarizeExternalSupportEmptySupportKeepsTopLevelWeakeningWarnings(t *testing.T) {
	tests := []struct {
		name     string
		evidence WebSearchEvidence
		warning  string
	}{
		{
			name: "no external docs",
			evidence: WebSearchEvidence{
				Enabled:      true,
				Truncated:    true,
				Inconclusive: true,
			},
			warning: "no_external_docs",
		},
		{
			name: "no citation-capable snippets",
			evidence: WebSearchEvidence{
				Enabled:      true,
				Truncated:    true,
				Inconclusive: true,
				ExternalDocs: []Evidence{
					{
						DocID:             "doc-1",
						URL:               "https://docs.example.test/doc-1",
						SourceCredibility: SourceCredibilityOfficialCandidate,
						FetchedAt:         time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
						Snippets: []SnippetEvidence{
							{SnippetID: "doc-1-snippet-1", Content: "missing hash"},
						},
					},
				},
			},
			warning: "no_citation_capable_snippets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := SummarizeExternalSupport(tt.evidence)

			if summary.Level != ExternalSupportLevelNone {
				t.Fatalf("Level = %q, want none", summary.Level)
			}
			if summary.OfficialConfirmation {
				t.Fatal("OfficialConfirmation = true, want false")
			}
			for _, want := range []string{tt.warning, "web_search_evidence_truncated", "web_search_evidence_inconclusive"} {
				if !slices.Contains(summary.Warnings, want) {
					t.Fatalf("Warnings = %#v, want %s", summary.Warnings, want)
				}
			}
		})
	}
}

func TestSummarizeExternalSupportSingleOfficialSnippetIsPartialNotStrong(t *testing.T) {
	summary := SummarizeExternalSupport(WebSearchEvidence{
		Enabled:      true,
		ExternalDocs: []Evidence{newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "only snippet")},
	})

	if summary.Level != ExternalSupportLevelPartial {
		t.Fatalf("Level = %q, want partial", summary.Level)
	}
	if summary.Level == ExternalSupportLevelStrong {
		t.Fatal("Level = strong, want conservative level")
	}
	if summary.OfficialConfirmation {
		t.Fatal("OfficialConfirmation = true, want false for partial support")
	}
	for _, want := range []string{"single_citation_capable_snippet_support", "single_official_candidate_source"} {
		if !slices.Contains(summary.Warnings, want) {
			t.Fatalf("Warnings = %#v, want %s", summary.Warnings, want)
		}
	}
}

func TestSummarizeExternalSupportUnknownAndThirdPartyOnlyAreWeak(t *testing.T) {
	tests := []struct {
		name        string
		credibility SourceCredibility
	}{
		{name: "unknown", credibility: SourceCredibilityUnknown},
		{name: "third-party", credibility: SourceCredibilityThirdParty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := SummarizeExternalSupport(WebSearchEvidence{
				Enabled:      true,
				ExternalDocs: []Evidence{newExternalSupportDocForTest("doc-1", tt.credibility, false, "snippet")},
			})

			if summary.Level != ExternalSupportLevelWeak {
				t.Fatalf("Level = %q, want weak", summary.Level)
			}
			if summary.OfficialConfirmation {
				t.Fatal("OfficialConfirmation = true, want false")
			}
			if !slices.Contains(summary.Warnings, "third_party_or_unknown_only_support") {
				t.Fatalf("Warnings = %#v, want third_party_or_unknown_only_support", summary.Warnings)
			}
		})
	}
}

func TestSummarizeExternalSupportOfficialWordingDoesNotCreateOfficialConfirmation(t *testing.T) {
	summary := SummarizeExternalSupport(WebSearchEvidence{
		Enabled: true,
		Queries: []WebSearchEvidenceQuery{
			{
				Query: "OpenAI API official documentation",
				Results: []WebSearchEvidenceResult{
					{
						Title:        "Official OpenAI API reference",
						URL:          "https://untrusted.example.test/reference",
						Snippet:      "Official documentation for OpenAI API requests.",
						SourceDomain: "untrusted.example.test",
					},
				},
			},
		},
		ExternalDocs: []Evidence{
			newExternalSupportDocForTest("doc-1", SourceCredibilityUnknown, false, "Official OpenAI API request documentation."),
		},
	})

	if summary.Level != ExternalSupportLevelWeak {
		t.Fatalf("Level = %q, want weak", summary.Level)
	}
	if summary.OfficialCandidateDocCount != 0 {
		t.Fatalf("OfficialCandidateDocCount = %d, want 0", summary.OfficialCandidateDocCount)
	}
	if summary.OfficialConfirmation {
		t.Fatal("OfficialConfirmation = true, want false")
	}
}

func TestSummarizeExternalSupportMultipleOfficialDocsIsAdequate(t *testing.T) {
	summary := SummarizeExternalSupport(WebSearchEvidence{
		Enabled: true,
		ExternalDocs: []Evidence{
			newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "first official snippet"),
			newExternalSupportDocForTest("doc-2", SourceCredibilityOfficialCandidate, false, "second official snippet"),
		},
	})

	if summary.Level != ExternalSupportLevelAdequate {
		t.Fatalf("Level = %q, want adequate", summary.Level)
	}
	if !summary.OfficialConfirmation {
		t.Fatal("OfficialConfirmation = false, want true")
	}
	if summary.OfficialCandidateCitationCapableDocCount != 2 {
		t.Fatalf("OfficialCandidateCitationCapableDocCount = %d, want 2", summary.OfficialCandidateCitationCapableDocCount)
	}
	if summary.OfficialCandidateUniqueCitationCapableSourceCount != 2 {
		t.Fatalf("OfficialCandidateUniqueCitationCapableSourceCount = %d, want 2", summary.OfficialCandidateUniqueCitationCapableSourceCount)
	}
	if summary.CitationCapableSnippetCount != 2 {
		t.Fatalf("CitationCapableSnippetCount = %d, want 2", summary.CitationCapableSnippetCount)
	}
}

func TestSummarizeExternalSupportDuplicateOfficialSourcesArePartial(t *testing.T) {
	tests := []struct {
		name   string
		adjust func([]Evidence)
	}{
		{
			name: "same normalized URL",
			adjust: func(docs []Evidence) {
				docs[1].URL = "https://docs.example.test/doc-1#section"
			},
		},
		{
			name: "same content hash",
			adjust: func(docs []Evidence) {
				docs[1].ContentHash = docs[0].ContentHash
			},
		},
		{
			name: "same snippet hash when doc hash is empty",
			adjust: func(docs []Evidence) {
				docs[0].ContentHash = ""
				docs[1].ContentHash = ""
				docs[1].Snippets[0].ContentHash = docs[0].Snippets[0].ContentHash
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			docs := []Evidence{
				newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "first official snippet"),
				newExternalSupportDocForTest("doc-2", SourceCredibilityOfficialCandidate, false, "second official snippet"),
			}
			tt.adjust(docs)

			summary := SummarizeExternalSupport(WebSearchEvidence{
				Enabled:      true,
				ExternalDocs: docs,
			})

			if summary.Level != ExternalSupportLevelPartial {
				t.Fatalf("Level = %q, want partial", summary.Level)
			}
			if summary.OfficialConfirmation {
				t.Fatal("OfficialConfirmation = true, want false")
			}
			if summary.OfficialCandidateCitationCapableDocCount != 2 {
				t.Fatalf("OfficialCandidateCitationCapableDocCount = %d, want raw doc count 2", summary.OfficialCandidateCitationCapableDocCount)
			}
			if summary.OfficialCandidateUniqueCitationCapableSourceCount != 1 {
				t.Fatalf("OfficialCandidateUniqueCitationCapableSourceCount = %d, want 1", summary.OfficialCandidateUniqueCitationCapableSourceCount)
			}
			if !slices.Contains(summary.Warnings, "duplicate_official_candidate_source") {
				t.Fatalf("Warnings = %#v, want duplicate_official_candidate_source", summary.Warnings)
			}
			if !slices.Contains(summary.Reasons, "level=partial: duplicate citation-capable official_candidate docs provide only one unique source") {
				t.Fatalf("Reasons = %#v, want duplicate source reason", summary.Reasons)
			}
		})
	}
}

func TestSummarizeExternalSupportWeakeningSignalsPreventAdequate(t *testing.T) {
	tests := []struct {
		name     string
		evidence WebSearchEvidence
		warning  string
	}{
		{
			name: "truncated",
			evidence: WebSearchEvidence{
				Enabled:   true,
				Truncated: true,
				ExternalDocs: []Evidence{
					newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "first official snippet"),
					newExternalSupportDocForTest("doc-2", SourceCredibilityOfficialCandidate, false, "second official snippet"),
				},
			},
			warning: "web_search_evidence_truncated",
		},
		{
			name: "inconclusive",
			evidence: WebSearchEvidence{
				Enabled:      true,
				Inconclusive: true,
				ExternalDocs: []Evidence{
					newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "first official snippet"),
					newExternalSupportDocForTest("doc-2", SourceCredibilityOfficialCandidate, false, "second official snippet"),
				},
			},
			warning: "web_search_evidence_inconclusive",
		},
		{
			name: "doc error",
			evidence: WebSearchEvidence{
				Enabled: true,
				ExternalDocs: []Evidence{
					newExternalSupportDocForTest("doc-1", SourceCredibilityOfficialCandidate, false, "first official snippet"),
					newExternalSupportDocForTest("doc-2", SourceCredibilityOfficialCandidate, false, "second official snippet"),
					{DocID: "doc-3", SourceCredibility: SourceCredibilityOfficialCandidate, Error: "fetch failed"},
				},
			},
			warning: "external_doc_error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			summary := SummarizeExternalSupport(tt.evidence)

			if summary.Level != ExternalSupportLevelPartial {
				t.Fatalf("Level = %q, want partial", summary.Level)
			}
			if summary.OfficialConfirmation {
				t.Fatal("OfficialConfirmation = true, want false")
			}
			if !slices.Contains(summary.Warnings, tt.warning) {
				t.Fatalf("Warnings = %#v, want %s", summary.Warnings, tt.warning)
			}
		})
	}
}

func newExternalSupportDocForTest(docID string, credibility SourceCredibility, truncated bool, snippets ...string) Evidence {
	doc := Evidence{
		DocID:             docID,
		URL:               "https://docs.example.test/" + docID,
		SourceDomain:      "docs.example.test",
		SourceCredibility: credibility,
		FetchedAt:         time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		Truncated:         truncated,
		ContentHash:       reviewExternalDocContentHash("doc:" + docID),
	}
	for i, content := range snippets {
		doc.Snippets = append(doc.Snippets, SnippetEvidence{
			SnippetID:   docID + "-snippet-" + strconv.Itoa(i+1),
			Content:     content,
			ContentHash: reviewExternalDocContentHash(content),
			Truncated:   truncated,
		})
	}
	return doc
}
