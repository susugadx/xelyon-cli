package report

import "testing"

func TestValidateReviewReportScopeCoverageSemanticContract(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "clean verdict rejects impact surface finding status",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceFinding
				return report
			},
			wantErr:     true,
			errContains: `verdict "clean" requires scope_coverage.reviewed_impact_surfaces[0].status`,
		},
		{
			name: "clean verdict rejects candidate risk finding status",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskFinding
				return report
			},
			wantErr:     true,
			errContains: `verdict "clean" requires scope_coverage.reviewed_candidate_risks[0].status`,
		},
		{
			name: "non-finding status cannot link finding IDs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = []string{"finding-1"}
				return report
			},
			wantErr:     true,
			errContains: "finding_ids must be empty when status is",
		},
		{
			name: "impact surface finding requires finding IDs",
			report: func() ReviewReport {
				return newImpactSurfaceFindingReportForValidationTest()
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].finding_ids must contain at least one root cause finding ID",
		},
		{
			name: "impact surface finding rejects unknown finding ID",
			report: func() ReviewReport {
				return newImpactSurfaceFindingReportForValidationTest("finding-unknown")
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].finding_ids[0] references unknown root cause finding ID",
		},
		{
			name: "impact surface finding requires evidence-backed root cause finding",
			report: func() ReviewReport {
				report := newBlockedImpactSurfaceFindingReportForValidationTest("finding-1")
				report.RootCauseGroups[0].Findings[0].EvidenceRefs = nil
				return report
			},
			wantErr:     true,
			errContains: `scope_coverage.reviewed_impact_surfaces[0].finding_ids[0] references root cause finding ID "finding-1" without evidence_refs`,
		},
		{
			name: "impact surface finding with evidence-backed root cause finding is valid",
			report: func() ReviewReport {
				return newImpactSurfaceFindingReportForValidationTest("finding-1")
			},
		},
		{
			name: "candidate risk finding requires finding IDs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskFinding
				return report
			},
			wantErr:     true,
			errContains: "finding_ids must contain at least one root cause finding ID",
		},
		{
			name: "root cause finding must be linked from scope coverage",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				return report
			},
			wantErr:     true,
			errContains: "must be referenced by scope_coverage finding_ids or new_findings_from_report_pass",
		},
	}

	runReviewReportValidationCases(t, tests)
}
