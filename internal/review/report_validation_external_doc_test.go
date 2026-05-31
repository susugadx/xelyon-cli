package review

import (
	"strings"
	"testing"
	"time"
)

func TestValidateReviewReportExternalDocEvidenceRefShape(t *testing.T) {
	report := newValidReviewReportForValidationTest()
	ref := newExternalDocEvidenceRefForValidationTest(newExternalDocEvidenceBundleForValidationTest())
	setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)

	if err := ValidateReviewReport(report); err != nil {
		t.Fatalf("ValidateReviewReport() error = %v, want nil", err)
	}

	report = newValidReviewReportForValidationTest()
	ref = newExternalDocEvidenceRefForValidationTest(newExternalDocEvidenceBundleForValidationTest())
	ref.ContentHash = "sha256:not-hex"
	setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
	if err := ValidateReviewReport(report); err == nil || !strings.Contains(err.Error(), "content_hash") {
		t.Fatalf("ValidateReviewReport() error = %v, want content_hash error", err)
	}

	report = newValidReviewReportForValidationTest()
	ref = newExternalDocEvidenceRefForValidationTest(newExternalDocEvidenceBundleForValidationTest())
	ref.Kind = "web_search"
	setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
	if err := ValidateReviewReport(report); err == nil || !strings.Contains(err.Error(), "kind") {
		t.Fatalf("ValidateReviewReport() error = %v, want raw web_search kind rejection", err)
	}
}

func TestFinalizeReviewRunnerReportAllowsFetchedExternalDocSnippetRef(t *testing.T) {
	bundle := newExternalDocEvidenceBundleForValidationTest()
	report := newRunnerCleanReportForTest(nil)
	ref := newExternalDocEvidenceRefForValidationTest(bundle)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}

	if _, err := finalizeReviewRunnerReport(report, newRunnerNoProbePlanForTest(), nil, newRunnerReportRedactorForTest(t, "/tmp/repo", nil), bundle); err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
}

func TestFinalizeReviewRunnerReportRejectsUnknownExternalDocSnippetRef(t *testing.T) {
	bundle := newExternalDocEvidenceBundleForValidationTest()
	report := newRunnerCleanReportForTest(nil)
	ref := newExternalDocEvidenceRefForValidationTest(bundle)
	ref.SnippetID = "external-doc-1-snippet-missing"
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}

	_, err := finalizeReviewRunnerReport(report, newRunnerNoProbePlanForTest(), nil, newRunnerReportRedactorForTest(t, "/tmp/repo", nil), bundle)
	if err == nil || !strings.Contains(err.Error(), "unknown fetched external_doc snippet") {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want unknown external_doc snippet error", err)
	}
}

func TestValidateReviewProbePlanAgainstEvidenceValidatesExternalDocRefs(t *testing.T) {
	bundle := newExternalDocEvidenceBundleForValidationTest()
	bundle.ChangedFiles = []ReviewChangedFile{{Path: "internal/api/providers/openai/web_search.go", Status: "M"}}
	bundle.Inventory = ReviewChangeInventory{Production: []string{"internal/api/providers/openai/web_search.go"}}
	bundle.RelatedSearchHits = []ReviewRelatedSearchHit{{Path: "internal/api/providers/openai/web_search_test.go", Line: 1, Snippet: "web_search", Reason: "test coverage"}}
	plan := newRunnerNoProbePlanForTest()
	plan.ImpactSurfaces[0].EvidenceSummary = "Production path internal/api/providers/openai/web_search.go and external_doc are covered."
	plan.ImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{newExternalDocEvidenceRefForValidationTest(bundle)}

	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err != nil {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want nil", err)
	}

	plan.ImpactSurfaces[0].EvidenceRefs[0].ContentHash = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	if err := ValidateReviewProbePlanAgainstEvidence(plan, bundle); err == nil || !strings.Contains(err.Error(), "content_hash") {
		t.Fatalf("ValidateReviewProbePlanAgainstEvidence() error = %v, want content_hash mismatch", err)
	}
}

func newExternalDocEvidenceBundleForValidationTest() ReviewEvidenceBundle {
	fetchedAt := time.Date(2026, time.May, 31, 12, 0, 0, 0, time.UTC)
	return ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   "/tmp/repo",
		CWD:        "/tmp/repo",
		WebSearchEvidence: ReviewWebSearchEvidence{
			Enabled:  true,
			Provider: "gemini",
			ExternalDocs: []ReviewExternalDocEvidence{
				{
					DocID:        "external-doc-1",
					URL:          "https://docs.example.test/spec",
					SourceDomain: "docs.example.test",
					FetchedAt:    fetchedAt,
					StatusCode:   200,
					ContentType:  "text/html",
					ContentHash:  "sha256:1111111111111111111111111111111111111111111111111111111111111111",
					Snippets: []ReviewExternalDocSnippetEvidence{
						{
							SnippetID:   "external-doc-1-snippet-1",
							Content:     "External spec text.",
							ContentHash: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
						},
					},
				},
			},
		},
	}
}

func newExternalDocEvidenceRefForValidationTest(bundle ReviewEvidenceBundle) ReviewEvidenceRef {
	doc := bundle.WebSearchEvidence.ExternalDocs[0]
	snippet := doc.Snippets[0]
	return ReviewEvidenceRef{
		Kind:        ReviewEvidenceKindExternalDoc,
		DocID:       doc.DocID,
		SnippetID:   snippet.SnippetID,
		URL:         doc.URL,
		FetchedAt:   doc.FetchedAt.Format(time.RFC3339Nano),
		ContentHash: snippet.ContentHash,
	}
}
