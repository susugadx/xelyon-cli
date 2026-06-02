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

func TestAuditReviewReportCoverageCalibratesMissingImpactSurfaceSeverity(t *testing.T) {
	tests := []struct {
		name       string
		status     PlanImpactSurfaceStatus
		want       CoverageIssueSeverity
		wantStatus ReviewSaturationStatus
	}{
		{name: "needs probe is high", status: PlanImpactSurfaceNeedsProbe, want: CoverageIssueSeverityHigh, wantStatus: ReviewSaturationStatusNeedsRevision},
		{name: "unverified is high", status: PlanImpactSurfaceUnverified, want: CoverageIssueSeverityHigh, wantStatus: ReviewSaturationStatusNeedsRevision},
		{name: "checked is medium", status: PlanImpactSurfaceChecked, want: CoverageIssueSeverityMedium, wantStatus: ReviewSaturationStatusSaturated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newNoProbePlanScopeForTest()
			plan.ImpactSurfaces[0].Status = tt.status
			report := newPlanAwareCleanReportForValidationTest()
			report.ScopeCoverage.ReviewedImpactSurfaces = nil

			issues := AuditReviewReportCoverage(CoverageAuditInput{
				Plan:   plan,
				Report: report,
			})

			assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindMissingImpactSurfaceCoverage, "surface-1", "", tt.want)
			merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
			if merged.Status != tt.wantStatus {
				t.Fatalf("merged status = %q, want %q", merged.Status, tt.wantStatus)
			}
		})
	}
}

func TestAuditReviewReportCoverageCalibratesMissingCandidateRiskSeverity(t *testing.T) {
	tests := []struct {
		name       string
		status     PlanCandidateRiskStatus
		severity   ReviewGroupSeverity
		want       CoverageIssueSeverity
		wantStatus ReviewSaturationStatus
	}{
		{name: "critical checked risk is high", status: PlanCandidateRiskCheckedByEvidence, severity: ReviewGroupSeverityCritical, want: CoverageIssueSeverityHigh, wantStatus: ReviewSaturationStatusNeedsRevision},
		{name: "high checked risk is high", status: PlanCandidateRiskCheckedByEvidence, severity: ReviewGroupSeverityHigh, want: CoverageIssueSeverityHigh, wantStatus: ReviewSaturationStatusNeedsRevision},
		{name: "needs probe risk is high", status: PlanCandidateRiskNeedsProbe, severity: ReviewGroupSeverityInfo, want: CoverageIssueSeverityHigh, wantStatus: ReviewSaturationStatusNeedsRevision},
		{name: "low checked risk is medium", status: PlanCandidateRiskCheckedByEvidence, severity: ReviewGroupSeverityLow, want: CoverageIssueSeverityMedium, wantStatus: ReviewSaturationStatusSaturated},
		{name: "info unverified risk is medium", status: PlanCandidateRiskUnverified, severity: ReviewGroupSeverityInfo, want: CoverageIssueSeverityMedium, wantStatus: ReviewSaturationStatusSaturated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan := newNoProbePlanScopeForTest()
			plan.CandidateRisks[0].Status = tt.status
			plan.CandidateRisks[0].Severity = tt.severity
			report := newPlanAwareCleanReportForValidationTest()
			report.ScopeCoverage.ReviewedCandidateRisks = nil

			issues := AuditReviewReportCoverage(CoverageAuditInput{
				Plan:   plan,
				Report: report,
			})

			assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindMissingCandidateRiskCoverage, "", "risk-1", tt.want)
			merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
			if merged.Status != tt.wantStatus {
				t.Fatalf("merged status = %q, want %q", merged.Status, tt.wantStatus)
			}
		})
	}
}

func TestAuditReviewReportCoverageDoesNotReportCandidateRiskHandledOutsideScopeCoverage(t *testing.T) {
	tests := []struct {
		name   string
		report func() ReviewReport
	}{
		{
			name: "finding text",
			report: func() ReviewReport {
				report := newPlanAwareHasFindingsReportForValidationTest()
				report.ScopeCoverage.ReviewedCandidateRisks = nil
				report.RootCauseGroups[0].Findings[0].Summary = "risk-1 is classified as a finding with file evidence."
				return report
			},
		},
		{
			name: "dismissed rationale",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedCandidateRisks = nil
				report.Summary = "risk-1 was dismissed because repository evidence covers the candidate."
				return report
			},
		},
		{
			name: "residual risk",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedCandidateRisks = nil
				report.ResidualRisks = []ReviewResidualRisk{{ID: "risk-1", Summary: "Keep monitoring the residual behavior."}}
				return report
			},
		},
		{
			name: "unverified rationale",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedCandidateRisks = nil
				report.Summary = "risk-1 remains unverified until a bounded probe can run."
				return report
			},
		},
		{
			name: "scope coverage",
			report: func() ReviewReport {
				return newPlanAwareCleanReportForValidationTest()
			},
		},
	}

	plan := newNoProbePlanScopeForTest()
	plan.CandidateRisks[0].Severity = ReviewGroupSeverityHigh
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := AuditReviewReportCoverage(CoverageAuditInput{
				Plan:   plan,
				Report: tt.report(),
			})

			assertNoCoverageIssueKindForTest(t, issues, CoverageIssueKindMissingCandidateRiskCoverage)
		})
	}
}
