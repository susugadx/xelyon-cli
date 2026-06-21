package externaldoc

import (
	"slices"
	"testing"
)

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
