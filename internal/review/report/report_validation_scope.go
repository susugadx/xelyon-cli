package report

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
