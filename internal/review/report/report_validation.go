package report

// ValidateReviewReport は schema v2 の review report 契約を検証する。
func ValidateReviewReport(report ReviewReport) error {
	if err := validateReportBasicFields(report); err != nil {
		return err
	}

	probeSummariesByID, err := validateProbeSummaries(report.ProbeSummaries)
	if err != nil {
		return err
	}

	if err := validateRootCauseGroups(report.RootCauseGroups); err != nil {
		return err
	}

	if err := validateReportRequiredContent(report); err != nil {
		return err
	}

	if err := validateEvidenceReferences(report, probeSummariesByID); err != nil {
		return err
	}

	if err := validateVerdictContract(report); err != nil {
		return err
	}

	if err := validateReviewReportScopeCoverageSemanticContract(report); err != nil {
		return err
	}

	return nil
}
