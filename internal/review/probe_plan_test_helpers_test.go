package review

import reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"

func markReviewProbePlanCheckedWithoutProbesForTest(plan reviewprobeplan.ReviewProbePlan) reviewprobeplan.ReviewProbePlan {
	plan.ImpactSurfaces[0].Status = reviewprobeplan.ReviewProbeImpactSurfaceChecked
	plan.ImpactSurfaces[0].Reason = "Existing evidence covers surface-1."
	plan.CandidateRisks[0].Status = reviewprobeplan.ReviewProbeCandidateRiskCheckedByEvidence
	plan.CandidateRisks[0].VerificationStrategy = "No probe is needed."
	plan.Probes = []reviewprobeplan.ReviewPlannedProbe{}
	plan.NoProbeReason = "surface-1 and risk-1 are checked by existing evidence."
	return plan
}
