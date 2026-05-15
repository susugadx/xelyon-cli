package review

import (
	"fmt"
	"strings"
)

// ValidateReviewSaturationCheck は runner 内部の final report saturation check 契約を検証する。
func ValidateReviewSaturationCheck(check ReviewSaturationCheck, plan ReviewProbePlan, finalizedReport ReviewReport) error {
	if check.SchemaVersion != ReviewSaturationCheckSchemaVersionV1 {
		return fmt.Errorf("schema_version must be %q: got %q", ReviewSaturationCheckSchemaVersionV1, check.SchemaVersion)
	}
	if !isKnownReviewSaturationStatus(check.Status) {
		return fmt.Errorf("status must be one of %q, %q, %q: got %q", ReviewSaturationStatusSaturated, ReviewSaturationStatusNeedsRevision, ReviewSaturationStatusBlocked, check.Status)
	}
	if strings.TrimSpace(check.CheckedSummary) == "" {
		return fmt.Errorf("checked_summary must be non-empty")
	}

	surfaceIDs := indexReviewSaturationImpactSurfaceIDs(plan)
	if err := validateReviewSaturationMissingIDs("missing_surface_ids", check.MissingSurfaceIDs, surfaceIDs, "impact surface"); err != nil {
		return err
	}
	riskIDs := indexReviewSaturationCandidateRiskIDs(plan)
	if err := validateReviewSaturationMissingIDs("missing_risk_ids", check.MissingRiskIDs, riskIDs, "candidate risk"); err != nil {
		return err
	}

	probeSummariesByID, err := validateProbeSummaries(finalizedReport.ProbeSummaries)
	if err != nil {
		return err
	}
	if err := validateReviewSaturationAdditionalFindingCandidates(check.AdditionalFindingCandidates, probeSummariesByID); err != nil {
		return err
	}

	switch check.Status {
	case ReviewSaturationStatusSaturated:
		return validateReviewSaturationSatisfiedCheck(check)
	case ReviewSaturationStatusNeedsRevision:
		return validateReviewSaturationNeedsRevisionCheck(check)
	case ReviewSaturationStatusBlocked:
		return nil
	default:
		return fmt.Errorf("status must be one of %q, %q, %q: got %q", ReviewSaturationStatusSaturated, ReviewSaturationStatusNeedsRevision, ReviewSaturationStatusBlocked, check.Status)
	}
}

func isKnownReviewSaturationStatus(status ReviewSaturationStatus) bool {
	switch status {
	case ReviewSaturationStatusSaturated, ReviewSaturationStatusNeedsRevision, ReviewSaturationStatusBlocked:
		return true
	default:
		return false
	}
}

func indexReviewSaturationImpactSurfaceIDs(plan ReviewProbePlan) map[string]struct{} {
	ids := make(map[string]struct{}, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		ids[surface.ID] = struct{}{}
	}
	return ids
}

func indexReviewSaturationCandidateRiskIDs(plan ReviewProbePlan) map[string]struct{} {
	ids := make(map[string]struct{}, len(plan.CandidateRisks))
	for _, risk := range plan.CandidateRisks {
		ids[risk.ID] = struct{}{}
	}
	return ids
}

func validateReviewSaturationMissingIDs(field string, ids []string, allowed map[string]struct{}, idKind string) error {
	seen := make(map[string]int, len(ids))
	for i, id := range ids {
		itemField := fmt.Sprintf("%s[%d]", field, i)
		canonicalID, err := validateReviewProbePlanID(itemField, id)
		if err != nil {
			return err
		}
		if _, exists := allowed[canonicalID]; !exists {
			return fmt.Errorf("%s references unknown Pass1 %s ID %q", itemField, idKind, canonicalID)
		}
		if firstIndex, exists := seen[canonicalID]; exists {
			return fmt.Errorf("%s duplicates %s ID %q first seen at %s[%d]", itemField, idKind, canonicalID, field, firstIndex)
		}
		seen[canonicalID] = i
	}
	return nil
}

func validateReviewSaturationAdditionalFindingCandidates(candidates []ReviewSaturationAdditionalFindingCandidate, probeSummariesByID map[string]ReviewProbeSummary) error {
	for i, candidate := range candidates {
		field := fmt.Sprintf("additional_finding_candidates[%d]", i)
		if strings.TrimSpace(candidate.Summary) == "" {
			return fmt.Errorf("%s.summary must be non-empty", field)
		}
		if strings.TrimSpace(candidate.Reason) == "" {
			return fmt.Errorf("%s.reason must be non-empty", field)
		}
		if len(candidate.EvidenceRefs) == 0 {
			return fmt.Errorf("%s.evidence_refs must contain at least one evidence ref", field)
		}
		if err := validateEvidenceRefs(field+".evidence_refs", candidate.EvidenceRefs, probeSummariesByID); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewSaturationSatisfiedCheck(check ReviewSaturationCheck) error {
	if len(check.MissingSurfaceIDs) > 0 {
		return fmt.Errorf("missing_surface_ids must be empty when status is %q", ReviewSaturationStatusSaturated)
	}
	if len(check.MissingRiskIDs) > 0 {
		return fmt.Errorf("missing_risk_ids must be empty when status is %q", ReviewSaturationStatusSaturated)
	}
	if len(check.AdditionalFindingCandidates) > 0 {
		return fmt.Errorf("additional_finding_candidates must be empty when status is %q", ReviewSaturationStatusSaturated)
	}
	if strings.TrimSpace(check.RevisionInstructions) != "" {
		return fmt.Errorf("revision_instructions must be empty when status is %q", ReviewSaturationStatusSaturated)
	}
	return nil
}

func validateReviewSaturationNeedsRevisionCheck(check ReviewSaturationCheck) error {
	if strings.TrimSpace(check.RevisionInstructions) == "" {
		return fmt.Errorf("revision_instructions must be non-empty when status is %q", ReviewSaturationStatusNeedsRevision)
	}
	if len(check.MissingSurfaceIDs) == 0 && len(check.MissingRiskIDs) == 0 && len(check.AdditionalFindingCandidates) == 0 {
		return fmt.Errorf("status %q requires missing_surface_ids, missing_risk_ids, or additional_finding_candidates", ReviewSaturationStatusNeedsRevision)
	}
	return nil
}
