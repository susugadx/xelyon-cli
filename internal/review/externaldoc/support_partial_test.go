package externaldoc

import (
	"slices"
	"testing"
)

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
