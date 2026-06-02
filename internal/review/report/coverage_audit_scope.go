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
			Severity:   missingImpactSurfaceCoverageSeverity(surface),
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
		if reportHandlesCandidateRiskOutsideScopeCoverage(input.Report, risk.ID) {
			continue
		}
		issues = append(issues, CoverageIssue{
			Kind:     CoverageIssueKindMissingCandidateRiskCoverage,
			Severity: missingCandidateRiskCoverageSeverity(risk),
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

func missingImpactSurfaceCoverageSeverity(surface PlanImpactSurface) CoverageIssueSeverity {
	switch surface.Status {
	case PlanImpactSurfaceNeedsProbe, PlanImpactSurfaceUnverified:
		return CoverageIssueSeverityHigh
	default:
		return CoverageIssueSeverityMedium
	}
}

func missingCandidateRiskCoverageSeverity(risk PlanCandidateRisk) CoverageIssueSeverity {
	if risk.Status == PlanCandidateRiskNeedsProbe {
		return CoverageIssueSeverityHigh
	}
	switch risk.Severity {
	case ReviewGroupSeverityCritical, ReviewGroupSeverityHigh:
		return CoverageIssueSeverityHigh
	default:
		return CoverageIssueSeverityMedium
	}
}

func reportHandlesCandidateRiskOutsideScopeCoverage(report ReviewReport, riskID string) bool {
	if strings.TrimSpace(riskID) == "" {
		return false
	}
	for _, risk := range report.ResidualRisks {
		if residualRiskHandlesCandidateRisk(risk, riskID) {
			return true
		}
	}
	if candidateRiskClassifiedInTexts(riskID, report.Summary) {
		return true
	}
	for _, group := range report.RootCauseGroups {
		if candidateRiskClassifiedInTexts(riskID, group.Title, group.Summary) {
			return true
		}
		for _, finding := range group.Findings {
			if candidateRiskClassifiedInTexts(riskID, finding.ID, finding.Title, finding.Summary) {
				return true
			}
			for _, risk := range finding.ResidualRisks {
				if residualRiskHandlesCandidateRisk(risk, riskID) {
					return true
				}
			}
		}
		for _, risk := range group.ResidualRisks {
			if residualRiskHandlesCandidateRisk(risk, riskID) {
				return true
			}
		}
	}
	return false
}

func residualRiskHandlesCandidateRisk(risk ReviewResidualRisk, riskID string) bool {
	return risk.ID == riskID || candidateRiskClassifiedInTexts(riskID, risk.Summary, risk.SuggestedMitigation)
}

func candidateRiskClassifiedInTexts(riskID string, texts ...string) bool {
	for _, text := range texts {
		if !coverageTextMentionsID(text, riskID) {
			continue
		}
		normalized := strings.ToLower(text)
		if strings.Contains(normalized, "dismissed") ||
			strings.Contains(normalized, "finding") ||
			strings.Contains(normalized, "residual") ||
			strings.Contains(normalized, "unverified") {
			return true
		}
	}
	return false
}

func coverageTextMentionsID(text, id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	searchFrom := 0
	for searchFrom <= len(text) {
		relativeIndex := strings.Index(text[searchFrom:], id)
		if relativeIndex < 0 {
			return false
		}
		start := searchFrom + relativeIndex
		end := start + len(id)
		if coverageTextHasIDBoundaryBefore(text, start) && coverageTextHasIDBoundaryAfter(text, end) {
			return true
		}
		searchFrom = start + 1
	}
	return false
}

func coverageTextHasIDBoundaryBefore(text string, index int) bool {
	if index <= 0 {
		return true
	}
	return !isReviewProbePlanIDRune(rune(text[index-1]))
}

func coverageTextHasIDBoundaryAfter(text string, index int) bool {
	if index >= len(text) {
		return true
	}
	return !isReviewProbePlanIDRune(rune(text[index]))
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
