package report

import "fmt"

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

func validateBlockedReviewReportScopeCoverageContract(report ReviewReport) error {
	if hasScopeCoverageUnverified(report.ScopeCoverage) || hasLegacyBlockedReason(report) {
		return nil
	}
	return fmt.Errorf("verdict %q requires unverified scope_coverage or blocked reason", ReviewVerdictBlocked)
}
