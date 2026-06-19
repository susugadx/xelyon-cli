package report

import "testing"

func TestValidateReviewReportScopeCoverageBaseContract(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "valid scope coverage with canonical finding IDs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = &ReviewReportScopeCoverage{
					ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
						{
							SurfaceID:  "surface-1",
							Status:     ReviewReportImpactSurfaceFinding,
							Summary:    "surface-1 is linked to finding-1.",
							FindingIDs: []string{"finding-1"},
						},
					},
				}
				return report
			},
		},
		{
			name: "invalid impact surface scope status",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].status",
		},
		{
			name: "invalid candidate risk scope status",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_candidate_risks[0].status",
		},
		{
			name: "scope coverage evidence refs are validated",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindFile}}
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "scope coverage finding IDs must be canonical",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = []string{"finding 1"}
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_candidate_risks[0].finding_ids[0]",
		},
		{
			name: "new report pass finding entry requires finding IDs",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.NewFindingsFromReportPass = []ReviewReportPassFindingCoverage{{Summary: "new finding"}}
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.new_findings_from_report_pass[0].finding_ids",
		},
	}

	runReviewReportValidationCases(t, tests)
}
