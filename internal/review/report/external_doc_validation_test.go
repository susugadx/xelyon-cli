package report

import (
	"strings"
	"testing"
	"time"
)

func TestValidateReviewReportExternalDocEvidenceRefShape(t *testing.T) {
	report := newValidReviewReportForValidationTest()
	ref := newExternalDocEvidenceRefForValidationTest()
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

func TestValidateExternalDocEvidenceRefsAcrossScopeCoverageAndSaturation(t *testing.T) {
	invalid := newExternalDocEvidenceRefForValidationTest()
	invalid.ContentHash = "sha256:not-hex"

	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.ScopeCoverage = &ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{{
			SurfaceID:    "surface-1",
			Status:       ReviewReportImpactSurfaceChecked,
			EvidenceRefs: []ReviewEvidenceRef{invalid},
		}},
		ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{{
			RiskID:       "risk-1",
			Status:       ReviewReportCandidateRiskDismissed,
			EvidenceRefs: []ReviewEvidenceRef{invalid},
		}},
		NewFindingsFromReportPass: []ReviewReportPassFindingCoverage{{
			FindingIDs:   []string{"finding-1"},
			EvidenceRefs: []ReviewEvidenceRef{invalid},
		}},
	}

	err := ValidateReviewReport(report)
	if err == nil || !strings.Contains(err.Error(), "scope_coverage.reviewed_impact_surfaces[0].evidence_refs[0].content_hash") {
		t.Fatalf("ValidateReviewReport() error = %v, want scope_coverage external_doc content_hash path", err)
	}

	check := ReviewSaturationCheck{
		SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
		Status:               ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "additional evidence checked",
		RevisionInstructions: "revise finding",
		AdditionalFindingCandidates: []ReviewSaturationAdditionalFindingCandidate{{
			Summary:      "candidate",
			Reason:       "unsupported external doc",
			EvidenceRefs: []ReviewEvidenceRef{invalid},
		}},
	}
	err = ValidateReviewSaturationCheck(check, newNoProbePlanScopeForTest(), newPlanAwareCleanReportForValidationTest())
	if err == nil || !strings.Contains(err.Error(), "additional_finding_candidates[0].evidence_refs[0].content_hash") {
		t.Fatalf("ValidateReviewSaturationCheck() error = %v, want saturation external_doc content_hash path", err)
	}
}

func newExternalDocEvidenceRefForValidationTest() ReviewEvidenceRef {
	return ReviewEvidenceRef{
		Kind:        ReviewEvidenceKindExternalDoc,
		DocID:       "external-doc-1",
		SnippetID:   "external-doc-1-snippet-1",
		URL:         "https://docs.example.test/spec",
		FetchedAt:   time.Date(2026, time.May, 31, 12, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		ContentHash: "sha256:2222222222222222222222222222222222222222222222222222222222222222",
	}
}
