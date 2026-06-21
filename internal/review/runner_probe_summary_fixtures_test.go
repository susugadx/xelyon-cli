package review

import (
	"testing"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewmodeloutput "github.com/susugadx/xelyon-cli/internal/review/modeloutput"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func newRedactedRunnerProbeSummariesForTest(t *testing.T, bundle reviewevidence.ReviewEvidenceBundle, results []reviewprobe.ReviewProbeResult) []reviewreport.ReviewProbeSummary {
	t.Helper()

	summaries := reviewprobe.BuildReviewProbeSummaries(results)
	probeIDs := make([]string, 0, len(summaries))
	for _, summary := range summaries {
		probeIDs = append(probeIDs, summary.ProbeID)
	}
	finalized, err := reviewmodeloutput.FinalizeReport(reviewmodeloutput.ReportFinalizationInput{
		Report:                newRunnerBlockedReportForTest(nil),
		Plan:                  newRunnerProbePlanForTest(probeIDs...),
		TrustedProbeSummaries: summaries,
		Redactor:              newReviewRunnerPromptRedactor(bundle, results),
	})
	if err != nil {
		t.Fatalf("FinalizeReport() error = %v, want nil", err)
	}
	return finalized.ProbeSummaries
}
