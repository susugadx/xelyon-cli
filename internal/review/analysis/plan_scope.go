package analysis

import (
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

// PlanScopeFromProbePlan は Pass1 probe plan から report validation 用の最小 scope を作る。
func PlanScopeFromProbePlan(plan reviewprobe.ReviewProbePlan) reviewreport.PlanScope {
	scope := reviewreport.PlanScope{
		ImpactSurfaces: make([]reviewreport.PlanImpactSurface, 0, len(plan.ImpactSurfaces)),
		CandidateRisks: make([]reviewreport.PlanCandidateRisk, 0, len(plan.CandidateRisks)),
		Probes:         make([]reviewreport.PlanProbe, 0, len(plan.Probes)),
	}
	for _, surface := range plan.ImpactSurfaces {
		scope.ImpactSurfaces = append(scope.ImpactSurfaces, reviewreport.PlanImpactSurface{
			ID:     surface.ID,
			Status: reviewreport.PlanImpactSurfaceStatus(surface.Status),
		})
	}
	for _, risk := range plan.CandidateRisks {
		scope.CandidateRisks = append(scope.CandidateRisks, reviewreport.PlanCandidateRisk{
			ID:     risk.ID,
			Status: reviewreport.PlanCandidateRiskStatus(risk.Status),
		})
	}
	for _, probe := range plan.Probes {
		scope.Probes = append(scope.Probes, reviewreport.PlanProbe{
			ID:         probe.ID,
			SurfaceIDs: append([]string(nil), probe.SurfaceIDs...),
			RiskIDs:    append([]string(nil), probe.RiskIDs...),
		})
	}
	return scope
}
