package review

import (
	"reflect"
	"strings"
	"testing"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func assertReviewReportDoesNotContainForRunnerTest(t *testing.T, report reviewreport.ReviewReport, leakedValues ...string) {
	t.Helper()

	reportJSON := string(mustMarshalReviewReportForRunnerTest(t, report))
	for _, leaked := range leakedValues {
		if leaked == "" {
			continue
		}
		if strings.Contains(reportJSON, leaked) {
			t.Fatalf("review report leaked %q:\n%s", leaked, reportJSON)
		}
	}
}

func assertStringSliceEqualForRunnerTest(t *testing.T, got, want []string) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("slice = %#v, want %#v", got, want)
	}
}

func assertReviewRunnerRequestPhasesForTest(t *testing.T, requests []ReviewModelRequest, want []ReviewModelPhase) {
	t.Helper()

	got := make([]ReviewModelPhase, 0, len(requests))
	for _, req := range requests {
		got = append(got, req.Phase)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("model request phases = %#v, want %#v", got, want)
	}
}
