package review

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewRunnerRunHappyPath(t *testing.T) {
	events := []string{}
	evidence := &runnerFakeEvidenceBuilder{
		bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo"),
		events: &events,
	}
	probeResult := reviewprobe.ReviewProbeResult{
		ID:              "probe-1",
		Mode:            reviewdomain.ReviewProbeHostReadOnly,
		Status:          reviewdomain.ReviewProbePassed,
		OutputTruncated: true,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command:         "go",
				Args:            []string{"test", "./internal/review"},
				Status:          reviewdomain.ReviewProbePassed,
				Output:          "PASS runner",
				OutputTruncated: true,
				Duration:        1500 * time.Millisecond,
			},
		},
	}
	probes := &runnerFakeProbeRunner{
		results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult},
		events:  &events,
	}
	plan := newRunnerProbePlanForTest("probe-1")
	probeResults := []reviewprobe.ReviewProbeResult{probeResult}
	modelReport := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")
	modelReport.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, probeResults)
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
			saturatedRunnerModelResponseForTest(t),
		},
		events: &events,
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest("focus on runner orchestration"))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	wantReport := withComputedSummaryForRunnerTest(modelReport, reviewprobe.BuildReviewProbeSummaries(probeResults))
	if !reflect.DeepEqual(got, wantReport) {
		t.Fatalf("Run() report = %#v, want %#v", got, wantReport)
	}
	assertStringSliceEqualForRunnerTest(t, events, []string{
		"evidence",
		"model:probe_plan",
		"probe:probe-1",
		"model:report",
		"model:saturation_check",
	})
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	if got, want := probes.calls[0].ID, "probe-1"; got != want {
		t.Fatalf("probe ID = %q, want %q", got, want)
	}
	if got, want := probes.calls[0].Commands[0].WorkDir, ""; got != want {
		t.Fatalf("probe command WorkDir = %q, want %q", got, want)
	}
	if got, want := len(model.requests), 3; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
	if got, want := model.requests[0].Phase, ReviewModelPhaseProbePlan; got != want {
		t.Fatalf("first model phase = %q, want %q", got, want)
	}
	if got, want := model.requests[1].Phase, ReviewModelPhaseReport; got != want {
		t.Fatalf("second model phase = %q, want %q", got, want)
	}
	if got, want := model.requests[2].Phase, ReviewModelPhaseSaturationCheck; got != want {
		t.Fatalf("third model phase = %q, want %q", got, want)
	}
	firstPrompt := model.requests[0].Prompt
	for _, want := range []string{"Review Pass 1", "focus on runner orchestration", "# Review Evidence", reviewprobeplan.ReviewProbePlanSchemaVersionV2} {
		if !strings.Contains(firstPrompt, want) {
			t.Fatalf("Pass1 prompt missing %q:\n%s", want, firstPrompt)
		}
	}
	secondPrompt := model.requests[1].Prompt
	for _, want := range []string{
		"Review Pass 2",
		reviewreport.ReviewReportSchemaVersionV2,
		"Probe Result Context",
		`"output": "PASS runner"`,
		`"status": "passed"`,
		`"output_truncated": true`,
		`"duration_ms": 1500`,
	} {
		if !strings.Contains(secondPrompt, want) {
			t.Fatalf("Pass2 prompt missing %q:\n%s", want, secondPrompt)
		}
	}
}

func TestReviewRunnerRunNoProbeReasonSkipsProbeRunner(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	modelReport := newRunnerCleanReportForTest([]reviewreport.ReviewProbeSummary{})
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	wantReport := withComputedSummaryForRunnerTest(modelReport, nil)
	if !reflect.DeepEqual(got, wantReport) {
		t.Fatalf("Run() report = %#v, want %#v", got, wantReport)
	}
	if got, want := len(probes.calls), 0; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	if got, want := len(model.requests), 3; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
	if !strings.Contains(model.requests[1].Prompt, `"no_probe_reason": "surface-1 and risk-1 are checked by existing evidence."`) {
		t.Fatalf("Pass2 prompt missing no_probe_reason:\n%s", model.requests[1].Prompt)
	}
}

func TestReviewRunnerRunsProbesInPlanOrder(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerProbePlanForTest("probe-a", "probe-b", "probe-c")
	results := []reviewprobe.ReviewProbeResult{
		{ID: "probe-a", Mode: reviewdomain.ReviewProbeHostReadOnly, Status: reviewdomain.ReviewProbePassed},
		{ID: "probe-b", Mode: reviewdomain.ReviewProbeHostReadOnly, Status: reviewdomain.ReviewProbePassed},
		{ID: "probe-c", Mode: reviewdomain.ReviewProbeHostReadOnly, Status: reviewdomain.ReviewProbePassed},
	}
	probes.results = map[string]reviewprobe.ReviewProbeResult{
		"probe-a": results[0],
		"probe-b": results[1],
		"probe-c": results[2],
	}
	report := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-a")
	report.ProbeSummaries = reviewprobe.BuildReviewProbeSummaries(results)
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	gotIDs := make([]string, 0, len(probes.calls))
	for _, call := range probes.calls {
		gotIDs = append(gotIDs, call.ID)
	}
	assertStringSliceEqualForRunnerTest(t, gotIDs, []string{"probe-a", "probe-b", "probe-c"})
}
