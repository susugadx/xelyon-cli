package report

import "fmt"

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
