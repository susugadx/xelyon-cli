package review

import (
	"fmt"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func finalizeReviewRunnerReport(report ReviewReport, plan ReviewProbePlan, trustedProbeSummaries []ReviewProbeSummary, redactor reviewRunnerPromptRedactor, bundle ReviewEvidenceBundle) (ReviewReport, error) {
	// LLM が返す probe_summaries は信頼元にしない。runner が probe results から作った
	// raw trusted summaries を内部 audit/debug 契約として保ち、final report には redacted copy だけを注入する。
	probeSummaries := reviewreport.CopyReviewProbeSummaries(trustedProbeSummaries)
	reviewreport.CanonicalizeReviewProbeSummaryMutationOutcomes(probeSummaries)
	if len(probeSummaries) == 0 {
		report.ProbeSummaries = nil
	} else {
		report.ProbeSummaries = redactReviewProbeSummaries(probeSummaries, redactor)
	}
	report = reviewreport.NormalizeReviewReportForTrustedProbeOutcomes(report)
	if err := ValidateReviewReportAgainstProbePlan(report, plan, trustedProbeSummaries); err != nil {
		return ReviewReport{}, fmt.Errorf("review runner finalize report: %w", err)
	}
	if err := validateReviewReportExternalDocRefsAgainstEvidence(report, bundle); err != nil {
		return ReviewReport{}, fmt.Errorf("review runner finalize report: %w", err)
	}
	computedSummary := ComputeReviewReportComputedSummary(report, probeSummaries)
	report.ComputedSummary = &computedSummary
	return report, nil
}
