package review

func newValidReviewProbePlanForTest() ReviewProbePlan {
	return ReviewProbePlan{
		SchemaVersion: ReviewProbePlanSchemaVersionV2,
		TargetKind:    TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Probe plan validation changed.",
				Category:        ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "Diff touches internal/review/probe_plan_validate.go.",
				Status:          ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Focused tests should verify the contract.",
			},
		},
		CandidateRisks: []ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "Validation could accept an invalid probe plan.",
				Severity:             ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Validation code owns the probe plan contract.",
				VerificationStrategy: "Run focused review tests.",
				Status:               ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: []ReviewPlannedProbe{
			{
				ID:             "probe-1",
				SurfaceIDs:     []string{"surface-1"},
				RiskIDs:        []string{"risk-1"},
				Purpose:        "Confirm or falsify risk-1 for surface-1 by running focused review tests.",
				Mode:           ReviewProbeRepoSandbox,
				TimeoutSeconds: 30,
				MaxOutputBytes: 4096,
				Commands: []ReviewPlannedProbeCommand{
					{
						Command: "go",
						Args:    []string{"test", "./internal/review"},
						WorkDir: ".",
					},
					{
						Command: "rg",
						Args:    []string{`path\like`, "ReviewProbePlan"},
						WorkDir: "internal/review",
					},
				},
				Files: []ReviewPlannedProbeFile{
					{
						Path:    "checks/check.txt",
						Content: "ok\n",
					},
				},
			},
		},
	}
}

func newNoProbeReviewProbePlanForTest() ReviewProbePlan {
	return markReviewProbePlanCheckedWithoutProbesForTest(newValidReviewProbePlanForTest())
}

func markReviewProbePlanCheckedWithoutProbesForTest(plan ReviewProbePlan) ReviewProbePlan {
	plan.ImpactSurfaces[0].Status = ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = []ReviewPlannedProbe{}
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}
