package report

import "testing"

func TestValidateReviewReportVerdictContract(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "clean with findings is invalid",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationVerified)
				report.RootCauseGroups = []ReviewRootCauseGroup{
					newRootCauseGroupForValidationTest("rc-1", "finding-1", ReviewVerificationVerified),
				}
				return report
			},
			wantErr:     true,
			errContains: "requires root_cause_groups to be empty",
		},
		{
			name: "clean with overall verified and no groups is valid",
			report: func() ReviewReport {
				return newCleanReportForValidationTest(ReviewVerificationVerified)
			},
		},
		{
			name: "clean with overall partially_verified and no groups is valid",
			report: func() ReviewReport {
				return newCleanReportForValidationTest(ReviewVerificationPartiallyVerified)
			},
		},
		{
			name: "clean with unverified surfaces is invalid",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationPartiallyVerified)
				report.UnverifiedSurfaces = []ReviewSurfaceCoverage{
					{SurfaceID: "surface-1", Summary: "surface-1 was not checked"},
				}
				return report
			},
			wantErr:     true,
			errContains: "unverified_surfaces",
		},
		{
			name: "clean with residual risks is invalid",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationPartiallyVerified)
				report.ResidualRisks = []ReviewResidualRisk{
					{Summary: "A residual risk remains unverified."},
				}
				return report
			},
			wantErr:     true,
			errContains: "residual_risks",
		},
		{
			name: "clean with overall unverified is invalid",
			report: func() ReviewReport {
				return newCleanReportForValidationTest(ReviewVerificationUnverified)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "clean with overall not_applicable is invalid",
			report: func() ReviewReport {
				return newCleanReportForValidationTest(ReviewVerificationNotApplicable)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "clean with overall blocked_or_inconclusive is invalid",
			report: func() ReviewReport {
				return newCleanReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "has_findings without groups is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups = nil
				return report
			},
			wantErr:     true,
			errContains: "requires at least one root_cause_group",
		},
		{
			name: "has_findings with empty groups is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups = []ReviewRootCauseGroup{}
				return report
			},
			wantErr:     true,
			errContains: "requires at least one root_cause_group",
		},
		{
			name: "has_findings with unverified group is invalid",
			report: func() ReviewReport {
				return newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationUnverified)
			},
			wantErr:     true,
			errContains: "verification_status",
		},
		{
			name: "has_findings with group without findings is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings = nil
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings",
		},
		{
			name: "has_findings with finding without evidence_refs is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].Findings[0].EvidenceRefs = nil
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].findings[0].evidence_refs",
		},
		{
			name: "has_findings with empty fix_strategy is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].FixStrategy = ""
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].fix_strategy",
		},
		{
			name: "has_findings with whitespace fix_strategy is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].FixStrategy = " \t\n"
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].fix_strategy",
		},
		{
			name: "has_findings with empty verification_plan is invalid",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.RootCauseGroups[0].VerificationPlan = nil
				return report
			},
			wantErr:     true,
			errContains: "root_cause_groups[0].verification_plan",
		},
		{
			name: "has_findings with evidence-backed group and overall verified is valid",
			report: func() ReviewReport {
				return newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
			},
		},
		{
			name: "has_findings with overall partially_verified is valid",
			report: func() ReviewReport {
				return newHasFindingsReportForValidationTest(ReviewVerificationPartiallyVerified, ReviewVerificationPartiallyVerified)
			},
		},
		{
			name: "has_findings with overall unverified is invalid",
			report: func() ReviewReport {
				return newHasFindingsReportForValidationTest(ReviewVerificationUnverified, ReviewVerificationVerified)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "has_findings with overall not_applicable is invalid",
			report: func() ReviewReport {
				return newHasFindingsReportForValidationTest(ReviewVerificationNotApplicable, ReviewVerificationVerified)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "has_findings with overall blocked_or_inconclusive is invalid",
			report: func() ReviewReport {
				return newHasFindingsReportForValidationTest(ReviewVerificationBlockedOrInconclusive, ReviewVerificationVerified)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "blocked with blocked probe summary is valid",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = ""
				report.ProbeSummaries = []ReviewProbeSummary{{
					ProbeID: "probe-1",
					Mode:    ReviewProbeHostReadOnly,
					Status:  ReviewProbeBlocked,
				}}
				return report
			},
		},
		{
			name: "blocked with timed_out probe summary is valid",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = ""
				report.ProbeSummaries = []ReviewProbeSummary{{
					ProbeID: "probe-1",
					Mode:    ReviewProbeScratchOnly,
					Status:  ReviewProbeTimedOut,
				}}
				return report
			},
		},
		{
			name: "blocked with mutated_worktree probe summary is valid",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = ""
				report.ProbeSummaries = []ReviewProbeSummary{{
					ProbeID: "probe-1",
					Mode:    ReviewProbeRepoSandbox,
					Status:  ReviewProbeMutatedWorktree,
				}}
				return report
			},
		},
		{
			name: "blocked with overall unverified is valid",
			report: func() ReviewReport {
				return newBlockedReportForValidationTest(ReviewVerificationUnverified)
			},
		},
		{
			name: "blocked with overall partially_verified is valid",
			report: func() ReviewReport {
				return newBlockedReportForValidationTest(ReviewVerificationPartiallyVerified)
			},
		},
		{
			name: "blocked with overall blocked_or_inconclusive is valid",
			report: func() ReviewReport {
				return newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
			},
		},
		{
			name: "blocked with overall verified is invalid",
			report: func() ReviewReport {
				return newBlockedReportForValidationTest(ReviewVerificationVerified)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "blocked with overall not_applicable is invalid",
			report: func() ReviewReport {
				return newBlockedReportForValidationTest(ReviewVerificationNotApplicable)
			},
			wantErr:     true,
			errContains: "overall_verification_status",
		},
		{
			name: "blocked with failed probe summary only is invalid",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = ""
				report.ProbeSummaries = []ReviewProbeSummary{{
					ProbeID: "probe-1",
					Mode:    ReviewProbeHostReadOnly,
					Status:  ReviewProbeFailed,
				}}
				return report
			},
			wantErr:     true,
			errContains: "requires blocked reason",
		},
		{
			name: "blocked with empty summary and no signals is invalid",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = "   "
				report.ProbeSummaries = nil
				return report
			},
			wantErr:     true,
			errContains: "requires blocked reason",
		},
		{
			name: "clean with checked surfaces and no groups is valid",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationPartiallyVerified)
				report.CheckedSurfaces = []ReviewSurfaceCoverage{
					{SurfaceID: "surface-1", Summary: "checked"},
				}
				return report
			},
		},
		{
			name: "suspected issue in residual risks is valid without findings",
			report: func() ReviewReport {
				report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
				report.Summary = ""
				report.ResidualRisks = []ReviewResidualRisk{{Summary: "未検証の境界条件が残る"}}
				return report
			},
		},
	}

	runReviewReportValidationCases(t, tests)
}
