package report

import "fmt"

func validateReviewReportScopeCoverageAgainstPlan(report ReviewReport, plan PlanScope) error {
	if report.ScopeCoverage == nil {
		return fmt.Errorf("scope_coverage is required when validating review report against probe plan")
	}
	if err := validateReviewReportImpactSurfaceCoverageAgainstPlan(report.ScopeCoverage, plan); err != nil {
		return err
	}
	if err := validateReviewReportCandidateRiskCoverageAgainstPlan(report.ScopeCoverage, plan); err != nil {
		return err
	}
	return nil
}

func validateReviewReportImpactSurfaceCoverageAgainstPlan(coverage *ReviewReportScopeCoverage, plan PlanScope) error {
	expected := make(map[string]struct{}, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		expected[surface.ID] = struct{}{}
	}

	seen := make(map[string]int, len(coverage.ReviewedImpactSurfaces))
	for i, surface := range coverage.ReviewedImpactSurfaces {
		_, exists := expected[surface.SurfaceID]
		if !exists {
			return fmt.Errorf("scope_coverage.reviewed_impact_surfaces[%d].surface_id references unknown impact surface ID %q", i, surface.SurfaceID)
		}
		if firstIndex, exists := seen[surface.SurfaceID]; exists {
			return fmt.Errorf("scope_coverage.reviewed_impact_surfaces[%d].surface_id duplicates impact surface ID %q first seen at reviewed_impact_surfaces[%d]", i, surface.SurfaceID, firstIndex)
		}
		seen[surface.SurfaceID] = i
	}
	for i, surface := range plan.ImpactSurfaces {
		if _, exists := seen[surface.ID]; !exists {
			return fmt.Errorf("scope_coverage.reviewed_impact_surfaces missing impact surface ID %q from impact_surfaces[%d]", surface.ID, i)
		}
	}
	return nil
}

func validateReviewReportCandidateRiskCoverageAgainstPlan(coverage *ReviewReportScopeCoverage, plan PlanScope) error {
	expected := make(map[string]struct{}, len(plan.CandidateRisks))
	for _, risk := range plan.CandidateRisks {
		expected[risk.ID] = struct{}{}
	}

	seen := make(map[string]int, len(coverage.ReviewedCandidateRisks))
	for i, risk := range coverage.ReviewedCandidateRisks {
		_, exists := expected[risk.RiskID]
		if !exists {
			return fmt.Errorf("scope_coverage.reviewed_candidate_risks[%d].risk_id references unknown candidate risk ID %q", i, risk.RiskID)
		}
		if firstIndex, exists := seen[risk.RiskID]; exists {
			return fmt.Errorf("scope_coverage.reviewed_candidate_risks[%d].risk_id duplicates candidate risk ID %q first seen at reviewed_candidate_risks[%d]", i, risk.RiskID, firstIndex)
		}
		seen[risk.RiskID] = i
	}
	for i, risk := range plan.CandidateRisks {
		if _, exists := seen[risk.ID]; !exists {
			return fmt.Errorf("scope_coverage.reviewed_candidate_risks missing candidate risk ID %q from candidate_risks[%d]", risk.ID, i)
		}
	}
	return nil
}
