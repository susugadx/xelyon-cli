package report

import "testing"

func TestAuditReviewReportCoverageReportsAdequateOfficialSupportAsHighWhenUnreflected(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked from repository context."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed from repository context."

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
		PostPass1ExternalEvidence: CoverageExternalEvidenceDelta{
			AddedQueryCount: 1,
			AddedQueries:    []string{"OAuth 2.0 redirect URI specification"},
			AddedDocIDs:     []string{"external-doc-official"},
			AddedDocURLs:    []string{"https://docs.example.test/oauth"},
		},
		ExternalSupport: CoverageExternalSupport{
			Level:                                "adequate",
			DocCount:                             2,
			CitationCapableDocCount:              2,
			CitationCapableSnippetCount:          2,
			OfficialCandidateDocCount:            2,
			OfficialCandidateCitationCapableDocs: 2,
			OfficialConfirmation:                 true,
		},
	})

	assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnreflectedExternalEvidence, "surface-1", "risk-1", CoverageIssueSeverityHigh)
	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
	if merged.Status != ReviewSaturationStatusNeedsRevision {
		t.Fatalf("merged Status = %q, want %q", merged.Status, ReviewSaturationStatusNeedsRevision)
	}
}
