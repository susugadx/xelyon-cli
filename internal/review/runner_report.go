package review

import "fmt"

func finalizeReviewRunnerReport(report ReviewReport, trustedProbeSummaries []ReviewProbeSummary) (ReviewReport, error) {
	// 現時点では raw trusted summaries を注入する。次フェーズで同じ core redaction helper を使い、
	// final report 用 redacted trusted summaries へ切り替える。
	report.ProbeSummaries = copyReviewRunnerProbeSummaries(trustedProbeSummaries)
	canonicalizeReviewProbeSummaryMutationOutcomes(report.ProbeSummaries)
	report = normalizeReviewRunnerReportForTrustedProbeOutcomes(report)
	if err := ValidateReviewReport(report); err != nil {
		return ReviewReport{}, fmt.Errorf("review runner finalize report: %w", err)
	}
	return report, nil
}

func normalizeReviewRunnerReportForTrustedProbeOutcomes(report ReviewReport) ReviewReport {
	if !hasReviewRunnerBlockedProbeSummary(report.ProbeSummaries) {
		return report
	}

	switch report.Verdict {
	case ReviewVerdictClean:
		report.Verdict = ReviewVerdictBlocked
		report.OverallVerificationStatus = ReviewVerificationBlockedOrInconclusive
	case ReviewVerdictHasFindings:
		if report.OverallVerificationStatus == ReviewVerificationVerified {
			report.OverallVerificationStatus = ReviewVerificationPartiallyVerified
		}
	case ReviewVerdictBlocked:
		if report.OverallVerificationStatus == ReviewVerificationVerified || report.OverallVerificationStatus == ReviewVerificationNotApplicable {
			report.OverallVerificationStatus = ReviewVerificationBlockedOrInconclusive
		}
	}
	return report
}

func hasReviewRunnerBlockedProbeSummary(summaries []ReviewProbeSummary) bool {
	for _, summary := range summaries {
		if isReviewProbeSummaryMutationOutcome(summary) {
			return true
		}
		switch summary.Status {
		case ReviewProbeBlocked, ReviewProbeTimedOut:
			return true
		}
	}
	return false
}

func copyReviewRunnerProbeSummaries(summaries []ReviewProbeSummary) []ReviewProbeSummary {
	if len(summaries) == 0 {
		return nil
	}

	copied := make([]ReviewProbeSummary, len(summaries))
	for i, summary := range summaries {
		copied[i] = summary
		copied[i].MutatedFiles = copyReviewRunnerStringSlice(summary.MutatedFiles)
		copied[i].Commands = copyReviewRunnerProbeCommandSummaries(summary.Commands)
	}
	return copied
}

func copyReviewRunnerProbeCommandSummaries(summaries []ReviewProbeCommandSummary) []ReviewProbeCommandSummary {
	if summaries == nil {
		return nil
	}

	copied := make([]ReviewProbeCommandSummary, len(summaries))
	for i, summary := range summaries {
		copied[i] = summary
		copied[i].Args = copyReviewRunnerStringSlice(summary.Args)
	}
	return copied
}

func copyReviewRunnerStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}
