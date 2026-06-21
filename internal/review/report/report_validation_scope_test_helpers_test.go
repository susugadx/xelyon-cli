package report

func newImpactSurfaceFindingReportForValidationTest(findingIDs ...string) ReviewReport {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	setImpactSurfaceFindingCoverageForValidationTest(&report, findingIDs...)
	return report
}

func newBlockedImpactSurfaceFindingReportForValidationTest(findingIDs ...string) ReviewReport {
	report := newBlockedReportForValidationTest(ReviewVerificationPartiallyVerified)
	report.RootCauseGroups = newRootCauseGroupsForValidationTest(ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	setImpactSurfaceFindingCoverageForValidationTest(&report, findingIDs...)
	return report
}

func setImpactSurfaceFindingCoverageForValidationTest(report *ReviewReport, findingIDs ...string) {
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceFinding
	report.ScopeCoverage.ReviewedImpactSurfaces[0].FindingIDs = append([]string(nil), findingIDs...)
}

func newPlanAwarePlanScopeForValidationTest() PlanScope {
	return newNoProbePlanScopeForTest()
}

func newPlanAwareCleanReportForValidationTest() ReviewReport {
	report := newCleanReportForValidationTest(ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	return report
}

func newPlanAwareBlockedReportForValidationTest() ReviewReport {
	report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	return report
}

func newPlanAwareBlockedReportWithRootCauseFindingForValidationTest() ReviewReport {
	report := newPlanAwareBlockedReportForValidationTest()
	report.RootCauseGroups = newRootCauseGroupsForValidationTest(ReviewVerificationVerified)
	return report
}

func newPlanAwareHasFindingsReportForValidationTest() ReviewReport {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.ScopeCoverage = &ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: ReviewReportImpactSurfaceChecked, Summary: "surface-1 was checked."},
		},
		ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
			{
				RiskID:     "risk-1",
				Status:     ReviewReportCandidateRiskFinding,
				Summary:    "risk-1 became finding-1.",
				FindingIDs: []string{"finding-1"},
			},
		},
	}
	return report
}
