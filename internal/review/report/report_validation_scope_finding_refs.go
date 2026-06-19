package report

import "fmt"

type reviewReportFindingInfo struct {
	evidenceBacked bool
}

func indexReviewReportRootCauseFindings(report ReviewReport) map[string]reviewReportFindingInfo {
	index := make(map[string]reviewReportFindingInfo)
	for _, group := range report.RootCauseGroups {
		for _, finding := range group.Findings {
			if finding.ID == "" {
				continue
			}
			index[finding.ID] = reviewReportFindingInfo{
				evidenceBacked: len(finding.EvidenceRefs) > 0,
			}
		}
	}
	return index
}

func validateReviewReportScopeCoverageFindingReferences(coverage *ReviewReportScopeCoverage, findingIndex map[string]reviewReportFindingInfo) error {
	for i, surface := range coverage.ReviewedImpactSurfaces {
		field := fmt.Sprintf("scope_coverage.reviewed_impact_surfaces[%d].finding_ids", i)
		if err := validateReviewReportScopeCoverageFindingIDReferences(field, surface.FindingIDs, findingIndex); err != nil {
			return err
		}
	}
	for i, risk := range coverage.ReviewedCandidateRisks {
		field := fmt.Sprintf("scope_coverage.reviewed_candidate_risks[%d].finding_ids", i)
		if err := validateReviewReportScopeCoverageFindingIDReferences(field, risk.FindingIDs, findingIndex); err != nil {
			return err
		}
	}
	for i, finding := range coverage.NewFindingsFromReportPass {
		field := fmt.Sprintf("scope_coverage.new_findings_from_report_pass[%d].finding_ids", i)
		if err := validateReviewReportScopeCoverageFindingIDReferences(field, finding.FindingIDs, findingIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewReportScopeCoverageFindingIDReferences(field string, findingIDs []string, findingIndex map[string]reviewReportFindingInfo) error {
	for i, findingID := range findingIDs {
		if _, exists := findingIndex[findingID]; !exists {
			return fmt.Errorf("%s[%d] references unknown root cause finding ID %q", field, i, findingID)
		}
	}
	return nil
}

func validateReviewReportScopeCoverageFindingIDStatusContract(coverage *ReviewReportScopeCoverage) error {
	for i, surface := range coverage.ReviewedImpactSurfaces {
		if len(surface.FindingIDs) == 0 {
			continue
		}
		if surface.Status != ReviewReportImpactSurfaceFinding {
			return fmt.Errorf("scope_coverage.reviewed_impact_surfaces[%d].finding_ids must be empty when status is %q", i, surface.Status)
		}
	}
	for i, risk := range coverage.ReviewedCandidateRisks {
		if len(risk.FindingIDs) == 0 {
			continue
		}
		if risk.Status != ReviewReportCandidateRiskFinding {
			return fmt.Errorf("scope_coverage.reviewed_candidate_risks[%d].finding_ids must be empty when status is %q", i, risk.Status)
		}
	}
	return nil
}

func validateRootCauseFindingScopeCoverageContract(report ReviewReport, findingIndex map[string]reviewReportFindingInfo) error {
	for i, group := range report.RootCauseGroups {
		for j, finding := range group.Findings {
			if finding.ID == "" {
				return fmt.Errorf("root_cause_groups[%d].findings[%d].id must be non-empty so scope_coverage can reference it", i, j)
			}
		}
	}
	return validateRootCauseFindingsLinkedFromScopeCoverage(report, findingIndex)
}

func validateImpactSurfaceFindingCoverage(coverage *ReviewReportScopeCoverage, findingIndex map[string]reviewReportFindingInfo) error {
	for i, surface := range coverage.ReviewedImpactSurfaces {
		if surface.Status != ReviewReportImpactSurfaceFinding {
			continue
		}
		field := fmt.Sprintf("scope_coverage.reviewed_impact_surfaces[%d]", i)
		if err := validateRequiredEvidenceBackedScopeCoverageFindingIDs(field, string(ReviewReportImpactSurfaceFinding), surface.FindingIDs, findingIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateCandidateRiskFindingCoverage(coverage *ReviewReportScopeCoverage, findingIndex map[string]reviewReportFindingInfo) error {
	for i, risk := range coverage.ReviewedCandidateRisks {
		if risk.Status != ReviewReportCandidateRiskFinding {
			continue
		}
		field := fmt.Sprintf("scope_coverage.reviewed_candidate_risks[%d]", i)
		if err := validateRequiredEvidenceBackedScopeCoverageFindingIDs(field, string(ReviewReportCandidateRiskFinding), risk.FindingIDs, findingIndex); err != nil {
			return err
		}
	}
	return nil
}

func validateRequiredEvidenceBackedScopeCoverageFindingIDs(field, status string, findingIDs []string, findingIndex map[string]reviewReportFindingInfo) error {
	if len(findingIDs) == 0 {
		return fmt.Errorf("%s.finding_ids must contain at least one root cause finding ID when status is %q", field, status)
	}
	for i, findingID := range findingIDs {
		info, exists := findingIndex[findingID]
		if !exists {
			// 不明な finding ID は validateReviewReportScopeCoverageFindingReferences が責務を持つ。
			continue
		}
		if !info.evidenceBacked {
			return fmt.Errorf("%s.finding_ids[%d] references root cause finding ID %q without evidence_refs", field, i, findingID)
		}
	}
	return nil
}

func validateRootCauseFindingsLinkedFromScopeCoverage(report ReviewReport, findingIndex map[string]reviewReportFindingInfo) error {
	linkedFindingIDs := collectReviewReportScopeCoverageFindingIDs(report.ScopeCoverage)
	for i, group := range report.RootCauseGroups {
		for j, finding := range group.Findings {
			if finding.ID == "" {
				continue
			}
			if _, exists := findingIndex[finding.ID]; !exists {
				continue
			}
			if _, exists := linkedFindingIDs[finding.ID]; !exists {
				return fmt.Errorf("root_cause_groups[%d].findings[%d].id %q must be referenced by scope_coverage finding_ids or new_findings_from_report_pass", i, j, finding.ID)
			}
		}
	}
	return nil
}

func collectReviewReportScopeCoverageFindingIDs(coverage *ReviewReportScopeCoverage) map[string]struct{} {
	linked := make(map[string]struct{})
	for _, surface := range coverage.ReviewedImpactSurfaces {
		if surface.Status != ReviewReportImpactSurfaceFinding {
			continue
		}
		for _, findingID := range surface.FindingIDs {
			linked[findingID] = struct{}{}
		}
	}
	for _, risk := range coverage.ReviewedCandidateRisks {
		if risk.Status != ReviewReportCandidateRiskFinding {
			continue
		}
		for _, findingID := range risk.FindingIDs {
			linked[findingID] = struct{}{}
		}
	}
	for _, finding := range coverage.NewFindingsFromReportPass {
		for _, findingID := range finding.FindingIDs {
			linked[findingID] = struct{}{}
		}
	}
	return linked
}
