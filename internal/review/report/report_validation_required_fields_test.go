package report

import "testing"

func TestValidateReviewReportRequiredNestedContent(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "finding title is required",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].Title = ""
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].title",
		},
		{
			name: "top-level checked surface id is required",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationPartiallyVerified)
				report.CheckedSurfaces = []ReviewSurfaceCoverage{{Summary: "checked"}}
				return report
			},
			wantErr:     true,
			errContains: "checked_surfaces[0].surface_id",
		},
		{
			name: "top-level unverified surface id is required",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = ""
				report.UnverifiedSurfaces = []ReviewSurfaceCoverage{{Summary: "blocked"}}
				return report
			},
			wantErr:     true,
			errContains: "unverified_surfaces[0].surface_id",
		},
		{
			name: "top-level residual risk summary is required",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = ""
				report.ResidualRisks = []ReviewResidualRisk{{}}
				return report
			},
			wantErr:     true,
			errContains: "residual_risks[0].summary",
		},
		{
			name: "group checked surface id is required",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].CheckedSurfaces = []ReviewSurfaceCoverage{{Summary: "checked"}}
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].checked_surfaces[0].surface_id",
		},
		{
			name: "group unverified surface id is required",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationPartiallyVerified, ReviewVerificationPartiallyVerified)
				report.RootCauseGroups[0].UnverifiedSurfaces = []ReviewSurfaceCoverage{{Summary: "unverified"}}
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].unverified_surfaces[0].surface_id",
		},
		{
			name: "group residual risk summary is required",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationPartiallyVerified, ReviewVerificationPartiallyVerified)
				report.RootCauseGroups[0].ResidualRisks = []ReviewResidualRisk{{}}
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].residual_risks[0].summary",
		},
		{
			name: "finding checked surface id is required",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].CheckedSurfaces = []ReviewSurfaceCoverage{{Summary: "checked"}}
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].checked_surfaces[0].surface_id",
		},
		{
			name: "finding unverified surface id is required",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationPartiallyVerified, ReviewVerificationPartiallyVerified)
				report.RootCauseGroups[0].Findings[0].UnverifiedSurfaces = []ReviewSurfaceCoverage{{Summary: "unverified"}}
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].unverified_surfaces[0].surface_id",
		},
		{
			name: "finding residual risk summary is required",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationPartiallyVerified, ReviewVerificationPartiallyVerified)
				report.RootCauseGroups[0].Findings[0].ResidualRisks = []ReviewResidualRisk{{}}
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].residual_risks[0].summary",
		},
	}

	runReviewReportValidationCases(t, tests)
}
