package probeplan

import "fmt"

type reviewProbePlanProbeLinkageValidator struct {
	surfaceIDs       map[string]struct{}
	riskIDs          map[string]struct{}
	linkedSurfaceIDs map[string]struct{}
	linkedRiskIDs    map[string]struct{}
}

func newReviewProbePlanProbeLinkageValidator(surfaceIDs, riskIDs map[string]struct{}, probeCount int) reviewProbePlanProbeLinkageValidator {
	return reviewProbePlanProbeLinkageValidator{
		surfaceIDs:       surfaceIDs,
		riskIDs:          riskIDs,
		linkedSurfaceIDs: make(map[string]struct{}, probeCount),
		linkedRiskIDs:    make(map[string]struct{}, probeCount),
	}
}

func (v reviewProbePlanProbeLinkageValidator) validateProbe(field string, probe ReviewPlannedProbe) error {
	if len(probe.SurfaceIDs) == 0 && len(probe.RiskIDs) == 0 {
		return fmt.Errorf("%s.surface_ids or %s.risk_ids must contain at least one referenced surface or risk ID", field, field)
	}
	for i, surfaceID := range probe.SurfaceIDs {
		refField := fmt.Sprintf("%s.surface_ids[%d]", field, i)
		canonicalSurfaceID, err := validateReviewProbePlanID(refField, surfaceID)
		if err != nil {
			return err
		}
		if _, exists := v.surfaceIDs[canonicalSurfaceID]; !exists {
			return fmt.Errorf("%s references unknown impact surface ID %q", refField, canonicalSurfaceID)
		}
		v.linkedSurfaceIDs[canonicalSurfaceID] = struct{}{}
	}
	for i, riskID := range probe.RiskIDs {
		refField := fmt.Sprintf("%s.risk_ids[%d]", field, i)
		canonicalRiskID, err := validateReviewProbePlanID(refField, riskID)
		if err != nil {
			return err
		}
		if _, exists := v.riskIDs[canonicalRiskID]; !exists {
			return fmt.Errorf("%s references unknown candidate risk ID %q", refField, canonicalRiskID)
		}
		v.linkedRiskIDs[canonicalRiskID] = struct{}{}
	}
	return nil
}

func (v reviewProbePlanProbeLinkageValidator) validateCoverage(surfaces []ReviewProbeImpactSurface, risks []ReviewProbeCandidateRisk) error {
	for i, surface := range surfaces {
		if surface.Status != ReviewProbeImpactSurfaceNeedsProbe && surface.Status != ReviewProbeImpactSurfaceUnverified {
			continue
		}
		if _, exists := v.linkedSurfaceIDs[surface.ID]; !exists {
			return fmt.Errorf("impact_surfaces[%d].id %q with status %q must be referenced by at least one probe surface_ids entry", i, surface.ID, surface.Status)
		}
	}
	for i, risk := range risks {
		if risk.Status != ReviewProbeCandidateRiskNeedsProbe && risk.Status != ReviewProbeCandidateRiskUnverified {
			continue
		}
		if _, exists := v.linkedRiskIDs[risk.ID]; !exists {
			return fmt.Errorf("candidate_risks[%d].id %q with status %q must be referenced by at least one probe risk_ids entry", i, risk.ID, risk.Status)
		}
	}
	return nil
}
