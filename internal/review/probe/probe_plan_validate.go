package probe

import "fmt"

// ValidateReviewProbePlan は LLM probe plan schema v2 の構造契約を検証する。
func ValidateReviewProbePlan(plan ReviewProbePlan) error {
	if plan.SchemaVersion != ReviewProbePlanSchemaVersionV2 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewProbePlanSchemaVersionV2, plan.SchemaVersion)
	}
	if plan.TargetKind != TargetCurrentChanges {
		return fmt.Errorf("target_kind must be %q: got %q", TargetCurrentChanges, plan.TargetKind)
	}
	surfaceIDs, err := validateReviewProbeImpactSurfaces(plan.ImpactSurfaces)
	if err != nil {
		return err
	}
	riskIDs, err := validateReviewProbeCandidateRisks(plan.CandidateRisks, surfaceIDs)
	if err != nil {
		return err
	}
	if err := validateReviewProbePlanNoCandidateRiskReason(plan, surfaceIDs); err != nil {
		return err
	}
	if len(plan.Probes) > MaxReviewProbePlanProbes {
		return fmt.Errorf("probes must contain at most %d entries: got %d", MaxReviewProbePlanProbes, len(plan.Probes))
	}
	if len(plan.Probes) == 0 {
		return validateReviewProbePlanNoProbeCompletion(plan, surfaceIDs, riskIDs)
	}
	if plan.NoProbeReason != "" {
		return fmt.Errorf("no_probe_reason must be empty when probes is non-empty")
	}

	seenIDs := make(map[string]struct{}, len(plan.Probes))
	linkage := newReviewProbePlanProbeLinkageValidator(surfaceIDs, riskIDs, len(plan.Probes))
	for i, probe := range plan.Probes {
		if err := validateReviewPlannedProbe(i, probe, seenIDs, linkage); err != nil {
			return err
		}
	}
	return linkage.validateCoverage(plan.ImpactSurfaces, plan.CandidateRisks)
}
