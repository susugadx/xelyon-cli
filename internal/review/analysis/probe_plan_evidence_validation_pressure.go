package analysis

import (
	"fmt"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func validateReviewProbePlanTruncationPressure(input EvidenceInput, plan reviewprobe.ReviewProbePlan) error {
	if !reviewEvidenceInputHasDiffContextOrSearchTruncation(input) || !reviewProbePlanAllImpactSurfacesChecked(plan) {
		return nil
	}
	return fmt.Errorf("impact_surfaces cannot all be checked when diff, context, or search evidence was truncated")
}

func validateReviewProbePlanGenericImpactTruncationPressure(input EvidenceInput, plan reviewprobe.ReviewProbePlan) error {
	if !input.GenericImpact.Truncated || len(plan.Probes) > 0 || !reviewProbePlanAllImpactSurfacesChecked(plan) {
		return nil
	}
	return fmt.Errorf("impact_surfaces cannot all be checked without probes when generic impact candidates were truncated")
}

func reviewEvidenceInputHasDiffContextOrSearchTruncation(input EvidenceInput) bool {
	for _, diff := range input.TruncationFlags.Diffs {
		if diff.Stat || diff.NameStatus || diff.Diff {
			return true
		}
	}
	for _, file := range input.TruncationFlags.ChangedFileContext {
		if file.Truncated {
			return true
		}
	}
	for _, file := range input.TruncationFlags.RelatedContextFiles {
		if file.Truncated {
			return true
		}
	}
	return input.TruncationFlags.RelatedCandidates || input.TruncationFlags.RelatedSearch
}

func validateReviewProbePlanNoProbeRequiresRelatedEvidence(input EvidenceInput, plan reviewprobe.ReviewProbePlan) error {
	if len(plan.Probes) > 0 || !reviewProbePlanAllImpactSurfacesChecked(plan) {
		return nil
	}
	if len(input.RelatedContextFiles) > 0 || len(input.RelatedSearchHits) > 0 {
		return nil
	}
	return fmt.Errorf("no-probe all-checked plan requires related context files or related search hits; absence of related evidence is not proof of no impact")
}

func reviewProbePlanAllImpactSurfacesChecked(plan reviewprobe.ReviewProbePlan) bool {
	if len(plan.ImpactSurfaces) == 0 {
		return false
	}
	for _, surface := range plan.ImpactSurfaces {
		if surface.Status != reviewprobe.ReviewProbeImpactSurfaceChecked {
			return false
		}
	}
	return true
}
