package report

func newValidPlanScopeForTest() PlanScope {
	return PlanScope{
		ImpactSurfaces: []PlanImpactSurface{
			{
				ID:     "surface-1",
				Status: PlanImpactSurfaceNeedsProbe,
			},
		},
		CandidateRisks: []PlanCandidateRisk{
			{
				ID:       "risk-1",
				Status:   PlanCandidateRiskNeedsProbe,
				Severity: ReviewGroupSeverityMedium,
			},
		},
		Probes: []PlanProbe{
			{
				ID:         "probe-1",
				SurfaceIDs: []string{"surface-1"},
				RiskIDs:    []string{"risk-1"},
			},
		},
	}
}

func newNoProbePlanScopeForTest() PlanScope {
	plan := newValidPlanScopeForTest()
	plan.ImpactSurfaces[0].Status = PlanImpactSurfaceChecked
	plan.CandidateRisks[0].Status = PlanCandidateRiskCheckedByEvidence
	plan.Probes = nil
	return plan
}

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
