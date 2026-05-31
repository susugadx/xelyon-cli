package report

import (
	"strings"
	"testing"
	"time"
)

func TestValidateReviewReportExternalDocEvidenceRefShape(t *testing.T) {
	report := newValidReviewReportForValidationTest()
	ref := ReviewEvidenceRef{
		Kind:        ReviewEvidenceKindExternalDoc,
		DocID:       "external-doc-1",
		SnippetID:   "external-doc-1-snippet-1",
		URL:         "https://docs.example.test/spec",
		FetchedAt:   time.Date(2026, time.May, 31, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		ContentHash: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
	}
	setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)

	if err := ValidateReviewReport(report); err != nil {
		t.Fatalf("ValidateReviewReport() error = %v, want nil", err)
	}

	report = newValidReviewReportForValidationTest()
	ref.ContentHash = "sha256:not-hex"
	setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
	if err := ValidateReviewReport(report); err == nil || !strings.Contains(err.Error(), "content_hash") {
		t.Fatalf("ValidateReviewReport() error = %v, want content_hash error", err)
	}

	report = newValidReviewReportForValidationTest()
	ref.ContentHash = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	ref.Kind = "web_search"
	setCheckedSurfaceEvidenceRefForValidationTest(&report, ref)
	if err := ValidateReviewReport(report); err == nil || !strings.Contains(err.Error(), "kind") {
		t.Fatalf("ValidateReviewReport() error = %v, want raw web_search kind rejection", err)
	}
}
