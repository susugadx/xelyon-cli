package modelinput

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func newValidReviewProbePlanForTest() reviewprobe.ReviewProbePlan {
	return reviewprobe.ReviewProbePlan{
		SchemaVersion: reviewprobe.ReviewProbePlanSchemaVersionV2,
		TargetKind:    domain.TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []reviewprobe.ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Probe plan validation changed.",
				Category:        reviewprobe.ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "Diff touches internal/review/probe_plan_validate.go.",
				Status:          reviewprobe.ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Focused tests should verify the contract.",
			},
		},
		CandidateRisks: []reviewprobe.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "Validation could accept an invalid probe plan.",
				Severity:             reviewreport.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Validation code owns the probe plan contract.",
				VerificationStrategy: "Run focused review tests.",
				Status:               reviewprobe.ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: []reviewprobe.ReviewPlannedProbe{
			{
				ID:             "probe-1",
				SurfaceIDs:     []string{"surface-1"},
				RiskIDs:        []string{"risk-1"},
				Purpose:        "Confirm or falsify risk-1 for surface-1 by running focused review tests.",
				Mode:           domain.ReviewProbeRepoSandbox,
				TimeoutSeconds: 30,
				MaxOutputBytes: 4096,
				Commands: []reviewprobe.ReviewPlannedProbeCommand{
					{
						Command: "go",
						Args:    []string{"test", "./internal/review"},
						WorkDir: ".",
					},
				},
			},
		},
	}
}

func newNoProbeReviewProbePlanForTest() reviewprobe.ReviewProbePlan {
	plan := newValidReviewProbePlanForTest()
	plan.ImpactSurfaces[0].Status = reviewprobe.ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = reviewprobe.ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = []reviewprobe.ReviewPlannedProbe{}
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}

func newNoProbePlanScopeForTest() reviewreport.PlanScope {
	return reviewreport.PlanScope{
		ImpactSurfaces: []reviewreport.PlanImpactSurface{
			{
				ID:     "surface-1",
				Status: reviewreport.PlanImpactSurfaceChecked,
			},
		},
		CandidateRisks: []reviewreport.PlanCandidateRisk{
			{
				ID:     "risk-1",
				Status: reviewreport.PlanCandidateRiskCheckedByEvidence,
			},
		},
	}
}

func newRunnerCleanReportForTest(probeSummaries []reviewreport.ReviewProbeSummary) reviewreport.ReviewReport {
	var reportProbeSummaries []reviewreport.ReviewProbeSummary
	if len(probeSummaries) > 0 {
		reportProbeSummaries = probeSummaries
	}
	return reviewreport.ReviewReport{
		SchemaVersion:             reviewreport.ReviewReportSchemaVersionV2,
		TargetKind:                domain.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: reviewreport.ReviewVerificationVerified,
		Verdict:                   reviewreport.ReviewVerdictClean,
		ProbeSummaries:            reportProbeSummaries,
		ScopeCoverage:             newCleanScopeCoverageForTest(),
	}
}

func withComputedSummaryForRunnerTest(report reviewreport.ReviewReport, probeSummaries []reviewreport.ReviewProbeSummary) reviewreport.ReviewReport {
	computedSummary := reviewreport.ComputeReviewReportComputedSummary(report, probeSummaries)
	report.ComputedSummary = &computedSummary
	return report
}

func newCleanScopeCoverageForTest() *reviewreport.ReviewReportScopeCoverage {
	return &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{
				SurfaceID: "surface-1",
				Status:    reviewreport.ReviewReportImpactSurfaceChecked,
				Summary:   "surface-1 was checked.",
			},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{
				RiskID:  "risk-1",
				Status:  reviewreport.ReviewReportCandidateRiskDismissed,
				Summary: "risk-1 was dismissed.",
			},
		},
	}
}

func newPlanAwareCleanReportForValidationTest() reviewreport.ReviewReport {
	return newRunnerCleanReportForTest(nil)
}
