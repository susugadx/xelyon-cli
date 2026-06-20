package analysis

import (
	"testing"

	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestPlanScopeFromProbePlanKeepsCandidateRiskSeverity(t *testing.T) {
	plan := reviewprobeplan.ReviewProbePlan{
		ImpactSurfaces: []reviewprobeplan.ReviewProbeImpactSurface{
			{
				ID:     "surface-1",
				Status: reviewprobeplan.ReviewProbeImpactSurfaceChecked,
			},
		},
		CandidateRisks: []reviewprobeplan.ReviewProbeCandidateRisk{
			{
				ID:       "risk-1",
				Status:   reviewprobeplan.ReviewProbeCandidateRiskCheckedByEvidence,
				Severity: reviewreport.ReviewGroupSeverityHigh,
			},
		},
	}

	scope := PlanScopeFromProbePlan(plan)

	if got, want := scope.CandidateRisks[0].Severity, reviewreport.ReviewGroupSeverityHigh; got != want {
		t.Fatalf("Severity = %q, want %q", got, want)
	}
}
