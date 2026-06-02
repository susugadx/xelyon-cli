package report

import (
	"fmt"
	"strings"
)

func auditMissingScopeCoverage(input CoverageAuditInput) []CoverageIssue {
	var issues []CoverageIssue
	coveredSurfaces := map[string]struct{}{}
	if input.Report.ScopeCoverage != nil {
		for _, surface := range input.Report.ScopeCoverage.ReviewedImpactSurfaces {
			coveredSurfaces[surface.SurfaceID] = struct{}{}
		}
	}
	for _, surface := range input.Plan.ImpactSurfaces {
		if _, ok := coveredSurfaces[surface.ID]; ok {
			continue
		}
		issues = append(issues, CoverageIssue{
			Kind:       CoverageIssueKindMissingImpactSurfaceCoverage,
			Severity:   CoverageIssueSeverityHigh,
			SurfaceIDs: []string{surface.ID},
			Summary:    fmt.Sprintf("Pass1 impact surface %q is missing from scope_coverage.", surface.ID),
			RevisionInstruction: fmt.Sprintf(
				"Add scope_coverage for impact surface %q and classify it as checked, finding, residual_risk, or unverified using the supplied evidence.",
				surface.ID,
			),
		})
	}

	coveredRisks := map[string]struct{}{}
	if input.Report.ScopeCoverage != nil {
		for _, risk := range input.Report.ScopeCoverage.ReviewedCandidateRisks {
			coveredRisks[risk.RiskID] = struct{}{}
		}
	}
	for _, risk := range input.Plan.CandidateRisks {
		if _, ok := coveredRisks[risk.ID]; ok {
			continue
		}
		issues = append(issues, CoverageIssue{
			Kind:     CoverageIssueKindMissingCandidateRiskCoverage,
			Severity: CoverageIssueSeverityHigh,
			RiskIDs:  []string{risk.ID},
			Summary:  fmt.Sprintf("Pass1 candidate risk %q is missing from scope_coverage.", risk.ID),
			RevisionInstruction: fmt.Sprintf(
				"Add scope_coverage for candidate risk %q and classify it as dismissed, finding, residual_risk, or unverified using the supplied evidence.",
				risk.ID,
			),
		})
	}
	return issues
}

func planScopeCoverageIDs(plan PlanScope) ([]string, []string) {
	surfaceIDs := make([]string, 0, len(plan.ImpactSurfaces))
	for _, surface := range plan.ImpactSurfaces {
		if id := strings.TrimSpace(surface.ID); id != "" {
			surfaceIDs = append(surfaceIDs, id)
		}
	}
	riskIDs := make([]string, 0, len(plan.CandidateRisks))
	for _, risk := range plan.CandidateRisks {
		if id := strings.TrimSpace(risk.ID); id != "" {
			riskIDs = append(riskIDs, id)
		}
	}
	return surfaceIDs, riskIDs
}
