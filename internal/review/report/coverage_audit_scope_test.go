package report

import "testing"

func TestAuditReviewReportCoverageNoIssueWhenScopeCoverageHandlesPass1(t *testing.T) {
	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: newPlanAwareCleanReportForValidationTest(),
	})

	if len(issues) != 0 {
		t.Fatalf("AuditReviewReportCoverage() issues = %#v, want none", issues)
	}
}

func TestAuditReviewReportCoverageReportsMissingScopeCoverage(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	report.ScopeCoverage.ReviewedImpactSurfaces = nil
	report.ScopeCoverage.ReviewedCandidateRisks = nil

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:   newNoProbePlanScopeForTest(),
		Report: report,
	})

	assertCoverageIssueForTest(t, issues, CoverageIssueKindMissingImpactSurfaceCoverage, "surface-1", "")
	assertCoverageIssueForTest(t, issues, CoverageIssueKindMissingCandidateRiskCoverage, "", "risk-1")
}
