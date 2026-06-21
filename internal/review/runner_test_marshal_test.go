package review

import (
	"encoding/json"
	"testing"

	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func mustMarshalReviewProbePlanForRunnerTest(t *testing.T, plan reviewprobeplan.ReviewProbePlan) []byte {
	t.Helper()

	data, err := json.Marshal(plan)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func mustMarshalReviewProbePlanWithMissingNoProbeReasonForRunnerTest(t *testing.T) []byte {
	t.Helper()

	plan := newRunnerNoProbePlanForTest()
	plan.NoProbeReason = ""
	return mustMarshalReviewProbePlanForRunnerTest(t, plan)
}

func mustMarshalReviewReportForRunnerTest(t *testing.T, report reviewreport.ReviewReport) []byte {
	t.Helper()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}
