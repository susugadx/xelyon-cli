package report

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
