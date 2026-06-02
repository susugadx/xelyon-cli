package analysis

import (
	"testing"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestPlanScopeFromProbePlanKeepsCandidateRiskSeverity(t *testing.T) {
	plan := reviewprobe.ReviewProbePlan{
		ImpactSurfaces: []reviewprobe.ReviewProbeImpactSurface{
			{
				ID:     "surface-1",
				Status: reviewprobe.ReviewProbeImpactSurfaceChecked,
			},
		},
		CandidateRisks: []reviewprobe.ReviewProbeCandidateRisk{
			{
				ID:       "risk-1",
				Status:   reviewprobe.ReviewProbeCandidateRiskCheckedByEvidence,
				Severity: reviewprobe.ReviewGroupSeverityHigh,
			},
		},
	}

	scope := PlanScopeFromProbePlan(plan)

	if got, want := scope.CandidateRisks[0].Severity, reviewreport.ReviewGroupSeverityHigh; got != want {
		t.Fatalf("Severity = %q, want %q", got, want)
	}
}
