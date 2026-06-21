package externaldoc

import (
	"slices"
	"testing"
)

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
