package report

import "fmt"

// ValidateReviewReportAgainstPlanScope は Pass2 report が Pass1 scope と trusted probe outcome を閉じていることを検証する。
func ValidateReviewReportAgainstPlanScope(report ReviewReport, plan PlanScope, trustedProbeSummaries []ReviewProbeSummary) error {
	if err := ValidateReviewReport(report); err != nil {
		return err
	}
	if trustedProbeSummaries != nil {
		if err := validateReviewReportTrustedProbeSummaries(report, trustedProbeSummaries); err != nil {
			return err
		}
	}
	if err := validateReviewReportScopeCoverageAgainstPlan(report, plan); err != nil {
		return err
	}
	if trustedProbeSummaries != nil {
		if err := validateReviewReportProbeOutcomesAgainstPlan(report, plan, trustedProbeSummaries); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewReportScopeCoverageShape(field string, coverage *ReviewReportScopeCoverage) error {
	if coverage == nil {
		return nil
	}
	for i, surface := range coverage.ReviewedImpactSurfaces {
		surfaceField := fmt.Sprintf("%s.reviewed_impact_surfaces[%d]", field, i)
		if _, err := validateRequiredReportID(surfaceField+".surface_id", surface.SurfaceID); err != nil {
			return err
		}
		if !isKnownReviewReportImpactSurfaceStatus(surface.Status) {
			return fmt.Errorf("%s.status must be known enum value: got %q", surfaceField, surface.Status)
		}
		if err := validateReviewReportFindingIDs(surfaceField+".finding_ids", surface.FindingIDs); err != nil {
			return err
		}
	}
	for i, risk := range coverage.ReviewedCandidateRisks {
		riskField := fmt.Sprintf("%s.reviewed_candidate_risks[%d]", field, i)
		if _, err := validateRequiredReportID(riskField+".risk_id", risk.RiskID); err != nil {
			return err
		}
		if !isKnownReviewReportCandidateRiskStatus(risk.Status) {
			return fmt.Errorf("%s.status must be known enum value: got %q", riskField, risk.Status)
		}
		if err := validateReviewReportFindingIDs(riskField+".finding_ids", risk.FindingIDs); err != nil {
			return err
		}
	}
	for i, finding := range coverage.NewFindingsFromReportPass {
		findingField := fmt.Sprintf("%s.new_findings_from_report_pass[%d]", field, i)
		if len(finding.FindingIDs) == 0 {
			return fmt.Errorf("%s.finding_ids must contain at least one finding ID", findingField)
		}
		if err := validateReviewReportFindingIDs(findingField+".finding_ids", finding.FindingIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewReportScopeCoverageSemanticContract(report ReviewReport) error {
	if report.ScopeCoverage == nil {
		return nil
	}
	if err := validateReviewReportScopeCoverageFindingIDStatusContract(report.ScopeCoverage); err != nil {
		return err
	}
	if err := validateReviewReportVerdictScopeCoverageContract(report); err != nil {
		return err
	}

	findingIndex := indexReviewReportRootCauseFindings(report)
	if err := validateReviewReportScopeCoverageFindingReferences(report.ScopeCoverage, findingIndex); err != nil {
		return err
	}
	if err := validateCandidateRiskFindingCoverage(report.ScopeCoverage, findingIndex); err != nil {
		return err
	}
	if err := validateImpactSurfaceFindingCoverage(report.ScopeCoverage, findingIndex); err != nil {
		return err
	}
	if err := validateRootCauseFindingScopeCoverageContract(report, findingIndex); err != nil {
		return err
	}
	return nil
}

func validateReviewReportFindingIDs(field string, findingIDs []string) error {
	for i, findingID := range findingIDs {
		if _, err := validateRequiredReportID(fmt.Sprintf("%s[%d]", field, i), findingID); err != nil {
			return err
		}
	}
	return nil
}

func validateReviewReportScopeCoverageEvidenceRefs(field string, coverage *ReviewReportScopeCoverage, probeSummariesByID map[string]ReviewProbeSummary) error {
	if coverage == nil {
		return nil
	}
	for i, surface := range coverage.ReviewedImpactSurfaces {
		if err := validateEvidenceRefs(fmt.Sprintf("%s.reviewed_impact_surfaces[%d].evidence_refs", field, i), surface.EvidenceRefs, probeSummariesByID); err != nil {
			return err
		}
	}
	for i, risk := range coverage.ReviewedCandidateRisks {
		if err := validateEvidenceRefs(fmt.Sprintf("%s.reviewed_candidate_risks[%d].evidence_refs", field, i), risk.EvidenceRefs, probeSummariesByID); err != nil {
			return err
		}
	}
	for i, finding := range coverage.NewFindingsFromReportPass {
		if err := validateEvidenceRefs(fmt.Sprintf("%s.new_findings_from_report_pass[%d].evidence_refs", field, i), finding.EvidenceRefs, probeSummariesByID); err != nil {
			return err
		}
	}
	return nil
}

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

func isKnownReviewReportImpactSurfaceStatus(status ReviewReportImpactSurfaceStatus) bool {
	for _, known := range reviewReportImpactSurfaceStatuses {
		if status == known {
			return true
		}
	}
	return false
}

func isKnownReviewReportCandidateRiskStatus(status ReviewReportCandidateRiskStatus) bool {
	for _, known := range reviewReportCandidateRiskStatuses {
		if status == known {
			return true
		}
	}
	return false
}

func hasScopeCoverageUnverified(coverage *ReviewReportScopeCoverage) bool {
	if coverage == nil {
		return false
	}
	for _, surface := range coverage.ReviewedImpactSurfaces {
		if surface.Status == ReviewReportImpactSurfaceUnverified {
			return true
		}
	}
	for _, risk := range coverage.ReviewedCandidateRisks {
		if risk.Status == ReviewReportCandidateRiskUnverified {
			return true
		}
	}
	return false
}

func validateReviewReportVerdictScopeCoverageContract(report ReviewReport) error {
	switch report.Verdict {
	case ReviewVerdictClean:
		return validateCleanReviewReportScopeCoverageContract(report.ScopeCoverage)
	case ReviewVerdictHasFindings:
		return nil
	case ReviewVerdictBlocked:
		return validateBlockedReviewReportScopeCoverageContract(report)
	default:
		return fmt.Errorf("verdict must be one of %q, %q, %q: got %q", ReviewVerdictClean, ReviewVerdictHasFindings, ReviewVerdictBlocked, report.Verdict)
	}
}

func validateCleanReviewReportScopeCoverageContract(coverage *ReviewReportScopeCoverage) error {
	for i, surface := range coverage.ReviewedImpactSurfaces {
		if surface.Status != ReviewReportImpactSurfaceChecked {
			return fmt.Errorf("verdict %q requires scope_coverage.reviewed_impact_surfaces[%d].status to be %q: got %q", ReviewVerdictClean, i, ReviewReportImpactSurfaceChecked, surface.Status)
		}
	}
	for i, risk := range coverage.ReviewedCandidateRisks {
		if risk.Status != ReviewReportCandidateRiskDismissed {
			return fmt.Errorf("verdict %q requires scope_coverage.reviewed_candidate_risks[%d].status to be %q: got %q", ReviewVerdictClean, i, ReviewReportCandidateRiskDismissed, risk.Status)
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

func validateBlockedReviewReportScopeCoverageContract(report ReviewReport) error {
	if hasScopeCoverageUnverified(report.ScopeCoverage) || hasLegacyBlockedReason(report) {
		return nil
	}
	return fmt.Errorf("verdict %q requires unverified scope_coverage or blocked reason", ReviewVerdictBlocked)
}
