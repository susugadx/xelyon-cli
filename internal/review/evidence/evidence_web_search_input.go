package evidence

import (
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
)

func buildReviewWebSearchQueryPlanningInput(bundle ReviewEvidenceBundle) externaldoc.SearchQueryPlanningInput {
	var parts []string
	for _, file := range bundle.ChangedFiles {
		parts = append(parts, file.Path, file.OldPath)
	}
	parts = append(parts, bundle.Inventory.Config...)
	parts = append(parts, bundle.Inventory.Production...)
	parts = append(parts, bundle.Inventory.Tests...)
	parts = append(parts, bundle.Inventory.Docs...)
	parts = append(parts, bundle.Inventory.Generated...)
	for _, diff := range bundle.Diffs {
		parts = append(parts, diff.Stat, diff.NameStatus, diff.Diff)
	}
	return externaldoc.SearchQueryPlanningInput{
		CorpusParts:         parts,
		GenericImpactTokens: append([]string(nil), bundle.GenericImpactCandidates.Tokens...),
	}
}

func buildReviewPostPass1WebSearchQueryPlanningInput(plan reviewprobeplan.ReviewProbePlan) externaldoc.SearchQueryPlanningInput {
	surfaces := make([]externaldoc.SearchQueryPlanImpactSurface, 0, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		surfaces = append(surfaces, externaldoc.SearchQueryPlanImpactSurface{
			ID:              surface.ID,
			Summary:         surface.Summary,
			Category:        string(surface.Category),
			EvidenceSummary: surface.EvidenceSummary,
			Reason:          surface.Reason,
		})
	}
	risks := make([]externaldoc.SearchQueryPlanCandidateRisk, 0, len(plan.CandidateRisks))
	for _, risk := range plan.CandidateRisks {
		risks = append(risks, externaldoc.SearchQueryPlanCandidateRisk{
			ID:                   risk.ID,
			Summary:              risk.Summary,
			Severity:             string(risk.Severity),
			SurfaceIDs:           append([]string(nil), risk.SurfaceIDs...),
			EvidenceSummary:      risk.EvidenceSummary,
			VerificationStrategy: risk.VerificationStrategy,
			Status:               string(risk.Status),
		})
	}
	return externaldoc.SearchQueryPlanningInput{
		ImpactSurfaces: surfaces,
		CandidateRisks: risks,
	}
}
