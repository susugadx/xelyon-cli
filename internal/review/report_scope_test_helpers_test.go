package review

func newCleanScopeCoverageForTest() *ReviewReportScopeCoverage {
	return &ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
			{
				SurfaceID: "surface-1",
				Status:    ReviewReportImpactSurfaceChecked,
				Summary:   "surface-1 was checked.",
			},
		},
		ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
			{
				RiskID:  "risk-1",
				Status:  ReviewReportCandidateRiskDismissed,
				Summary: "risk-1 was dismissed.",
			},
		},
	}
}
