package review

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestReviewRunnerRunInvalidPass1PlanStopsBeforeProbesAndPass2(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{
			name:    "invalid JSON",
			content: `{not-json`,
		},
		{
			name:    "invalid validated plan",
			content: string(mustMarshalReviewProbePlanWithMissingNoProbeReasonForRunnerTest(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
			probes := &runnerFakeProbeRunner{}
			model := &runnerFakeModel{
				responses: []runnerFakeModelResponse{
					{content: tt.content},
					{content: tt.content},
					{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
					{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
				},
			}
			runner := newReviewRunnerForTest(t, evidence, probes, model)

			_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
			if err == nil {
				t.Fatal("Run() error = nil, want error")
			}
			if !strings.Contains(err.Error(), "decode probe plan") {
				t.Fatalf("Run() error = %q, want decode probe plan", err.Error())
			}
			if got, want := len(probes.calls), 0; got != want {
				t.Fatalf("probe calls = %d, want %d", got, want)
			}
			if got, want := len(model.requests), 2; got != want {
				t.Fatalf("model requests = %d, want %d", got, want)
			}
			if got, want := model.requests[1].Phase, ReviewModelPhaseProbePlan; got != want {
				t.Fatalf("repair model phase = %q, want %q", got, want)
			}
		})
	}
}

func TestReviewRunnerRunInvalidPass2ReportReturnsAfterProbeExecution(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: `{"schema_version":"review_report.v2"}`},
			{content: `{"schema_version":"review_report.v2"}`},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "finalize report") {
		t.Fatalf("Run() error = %q, want finalize report", err.Error())
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	if got, want := len(model.requests), 3; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
	if got, want := model.requests[2].Phase, ReviewModelPhaseReport; got != want {
		t.Fatalf("repair model phase = %q, want %q", got, want)
	}
}

func TestReviewRunnerRunEvidenceBuildErrorStopsBeforeModel(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{err: errors.New("git status failed")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "git status failed") {
		t.Fatalf("Run() error = %q, want evidence error", err.Error())
	}
	if got, want := len(model.requests), 0; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
}

func TestReviewRunnerRunPass1ModelErrorStopsBeforeProbesAndPass2(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{responses: []runnerFakeModelResponse{{err: errors.New("model unavailable")}}}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "pass1 model") {
		t.Fatalf("Run() error = %q, want pass1 model", err.Error())
	}
	if got, want := len(probes.calls), 0; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	if got, want := len(model.requests), 1; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
}

func TestReviewRunnerRunPass2ModelErrorReturnsAfterProbeExecution(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{err: errors.New("report model failed")},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "pass2 model") {
		t.Fatalf("Run() error = %q, want pass2 model", err.Error())
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	if got, want := len(model.requests), 2; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
}

func TestReviewRunnerRunProbeExecutorErrorStopsBeforePass2(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{errors: map[string]error{"probe-1": errors.New("sandbox setup failed")}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "sandbox setup failed") {
		t.Fatalf("Run() error = %q, want probe error", err.Error())
	}
	if got, want := len(model.requests), 1; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
}
