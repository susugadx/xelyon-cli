package probeplan

import (
	"fmt"
	"strings"
)

func validateReviewProbeImpactSurfaces(surfaces []ReviewProbeImpactSurface) (map[string]struct{}, error) {
	if len(surfaces) == 0 {
		return nil, fmt.Errorf("impact_surfaces must contain at least one entry")
	}

	seenIDs := make(map[string]struct{}, len(surfaces))
	for i, surface := range surfaces {
		field := fmt.Sprintf("impact_surfaces[%d]", i)
		id, err := validateReviewProbePlanID(field+".id", surface.ID)
		if err != nil {
			return nil, err
		}
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("%s.id duplicates %q", field, id)
		}
		seenIDs[id] = struct{}{}

		if err := validateReviewProbePlanRequiredText(field+".summary", surface.Summary); err != nil {
			return nil, err
		}
		if err := validateReviewProbePlanRequiredText(field+".reason", surface.Reason); err != nil {
			return nil, err
		}
		if !isKnownReviewProbeImpactSurfaceCategory(surface.Category) {
			return nil, fmt.Errorf("%s.category must be known enum value: got %q", field, surface.Category)
		}
		if !isKnownReviewProbeImpactSurfaceStatus(surface.Status) {
			return nil, fmt.Errorf("%s.status must be known enum value: got %q", field, surface.Status)
		}
		if err := validateReviewProbePlanPreProbeEvidence(field, surface.EvidenceSummary, surface.EvidenceRefs); err != nil {
			return nil, err
		}
	}
	return seenIDs, nil
}

func validateReviewProbeCandidateRisks(risks []ReviewProbeCandidateRisk, surfaceIDs map[string]struct{}) (map[string]struct{}, error) {
	seenIDs := make(map[string]struct{}, len(risks))
	for i, risk := range risks {
		field := fmt.Sprintf("candidate_risks[%d]", i)
		id, err := validateReviewProbePlanID(field+".id", risk.ID)
		if err != nil {
			return nil, err
		}
		if _, exists := seenIDs[id]; exists {
			return nil, fmt.Errorf("%s.id duplicates %q", field, id)
		}
		seenIDs[id] = struct{}{}

		if err := validateReviewProbePlanRequiredText(field+".summary", risk.Summary); err != nil {
			return nil, err
		}
		if err := validateReviewProbePlanRequiredText(field+".verification_strategy", risk.VerificationStrategy); err != nil {
			return nil, err
		}
		if !isKnownReviewGroupSeverity(risk.Severity) {
			return nil, fmt.Errorf("%s.severity must be known enum value: got %q", field, risk.Severity)
		}
		if !isKnownReviewProbeCandidateRiskStatus(risk.Status) {
			return nil, fmt.Errorf("%s.status must be known enum value: got %q", field, risk.Status)
		}
		if len(risk.SurfaceIDs) == 0 {
			return nil, fmt.Errorf("%s.surface_ids must contain at least one impact surface ID", field)
		}
		for j, surfaceID := range risk.SurfaceIDs {
			refField := fmt.Sprintf("%s.surface_ids[%d]", field, j)
			canonicalSurfaceID, err := validateReviewProbePlanID(refField, surfaceID)
			if err != nil {
				return nil, err
			}
			if _, exists := surfaceIDs[canonicalSurfaceID]; !exists {
				return nil, fmt.Errorf("%s references unknown impact surface ID %q", refField, canonicalSurfaceID)
			}
		}
		if err := validateReviewProbePlanPreProbeEvidence(field, risk.EvidenceSummary, risk.EvidenceRefs); err != nil {
			return nil, err
		}
	}
	return seenIDs, nil
}

func validateReviewProbePlanPreProbeEvidence(field, evidenceSummary string, refs []ReviewEvidenceRef) error {
	if strings.TrimSpace(evidenceSummary) == "" && len(refs) == 0 {
		return fmt.Errorf("%s requires evidence_summary or evidence_refs", field)
	}
	return validateReviewProbePlanPreProbeEvidenceRefs(field+".evidence_refs", refs)
}

func validateReviewProbePlanPreProbeEvidenceRefs(field string, refs []ReviewEvidenceRef) error {
	for i, ref := range refs {
		refField := fmt.Sprintf("%s[%d]", field, i)
		if !isReviewProbePlanPreProbeEvidenceKind(ref.Kind) {
			if !isKnownReviewEvidenceKind(ref.Kind) {
				return validateEvidenceRef(refField, ref, nil)
			}
			return fmt.Errorf("%s.kind must reference pre-probe evidence, got %q", refField, ref.Kind)
		}
		if ref.ProbeID != "" {
			return fmt.Errorf("%s.probe_id is not allowed in probe plan pre-probe evidence", refField)
		}
		if ref.CommandIndex != nil {
			return fmt.Errorf("%s.command_index is not allowed in probe plan pre-probe evidence", refField)
		}
		if err := validateEvidenceRef(refField, ref, nil); err != nil {
			return err
		}
	}
	return nil
}
