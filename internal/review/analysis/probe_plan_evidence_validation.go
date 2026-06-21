package analysis

import reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"

// ValidateProbePlanAgainstEvidence は Pass1 probe plan が evidence input の
// material path と evidence pressure を扱っていることを検証する。
func ValidateProbePlanAgainstEvidence(plan reviewprobeplan.ReviewProbePlan, input EvidenceInput) error {
	if err := reviewprobeplan.ValidateReviewProbePlan(plan); err != nil {
		return err
	}
	if err := ValidateProbePlanExternalDocRefs(plan, input.WebSearchEvidence.ExternalDocs); err != nil {
		return err
	}

	index := newReviewProbePlanImpactSurfaceEvidenceIndex(plan.ImpactSurfaces)

	if err := validateReviewProbePlanMaterialPathCoverage(input, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanInventoryCategoryCoverage(input, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanUntrackedCoverage(input, plan, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanGenericImpactCoverage(input, index); err != nil {
		return err
	}
	if err := validateReviewProbePlanTruncationPressure(input, plan); err != nil {
		return err
	}
	if err := validateReviewProbePlanGenericImpactTruncationPressure(input, plan); err != nil {
		return err
	}
	if err := validateReviewProbePlanNoProbeRequiresRelatedEvidence(input, plan); err != nil {
		return err
	}
	return nil
}
