package externaldoc

import (
	"slices"
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
