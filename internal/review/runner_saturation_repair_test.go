package review

import (
	"context"
	"reflect"
	"strings"
	"testing"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewRunnerRunRepairsInvalidSaturationJSONOnce(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
			{content: `{not-json`},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got.Verdict != reviewreport.ReviewVerdictClean {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, reviewreport.ReviewVerdictClean)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseSaturationCheck,
	})
	for _, want := range []string{"Saturation Check JSON Repair", "Return corrected JSON only.", "{not-json", "decode review saturation check"} {
		if !strings.Contains(model.requests[3].Prompt, want) {
			t.Fatalf("saturation repair prompt missing %q:\n%s", want, model.requests[3].Prompt)
		}
	}
}

func TestReviewRunnerRunInvalidSaturationRepairReturnsError(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
			{content: `{not-json`},
			{content: `{still-not-json`},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want saturation decode error")
	}
	if !strings.Contains(err.Error(), "decode saturation check") {
		t.Fatalf("Run() error = %q, want saturation decode error", err.Error())
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseSaturationCheck,
	})
}

func TestReviewRunnerRunBlockedSaturationReturnsError(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	blockedCheck := reviewreport.ReviewSaturationCheck{
		SchemaVersion:  reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:         reviewreport.ReviewSaturationStatusBlocked,
		CheckedSummary: "Probe result context was insufficient to check saturation.",
	}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, blockedCheck))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want blocked saturation error")
	}
	for _, want := range []string{"saturation check blocked", "insufficient to check saturation"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Run() error = %q, want %q", err.Error(), want)
		}
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
}

func TestReviewRunnerRunRepairsInvalidReportRevisionJSONOnce(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	revisedReport := newPlanAwareHasFindingsReportForValidationTest()
	revisedReport.ProbeSummaries = nil
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, needsRevisionMissingRiskCheckForRunnerTest()))},
			{content: `{"schema_version":"review_report.v2"}`},
			{content: string(mustMarshalReviewReportForRunnerTest(t, revisedReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	want := withComputedSummaryForRunnerTest(revisedReport, nil)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Run() report = %#v, want repaired revision %#v", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseSaturationCheck,
	})
	repairPrompt := model.requests[4].Prompt
	for _, wantText := range []string{
		"Review Pass 2: Report Revision JSON Repair",
		"## Invalid Model Output",
		`{"schema_version":"review_report.v2"}`,
		"## Decode Or Validation Error",
		"target_kind",
	} {
		if !strings.Contains(repairPrompt, wantText) {
			t.Fatalf("revision repair prompt missing %q:\n%s", wantText, repairPrompt)
		}
	}
}

func TestReviewRunnerRunInvalidReportRevisionRepairReturnsError(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, needsRevisionMissingRiskCheckForRunnerTest()))},
			{content: `{"schema_version":"review_report.v2"}`},
			{content: `{"schema_version":"review_report.v2"}`},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want report revision repair error")
	}
	for _, want := range []string{"report revision repair", "target_kind"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Run() error = %q, want %q", err.Error(), want)
		}
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseReportRevision,
	})
}
