package review

import (
	"context"
	"reflect"
	"strings"
	"testing"

	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewRunnerRunRevisesReportWhenSaturationNeedsMissingRisk(t *testing.T) {
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
		t.Fatalf("Run() report = %#v, want revised %#v", got, want)
	}
	if got.ComputedSummary == nil || got.ComputedSummary.FindingCount != 1 || got.ComputedSummary.FindingRiskCount != 1 {
		t.Fatalf("ComputedSummary = %#v, want recalculated finding counts", got.ComputedSummary)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseSaturationCheck,
	})
	revisionPrompt := model.requests[3].Prompt
	for _, wantText := range []string{
		"Review Pass 2: Report Revision",
		"## Saturation Check",
		`"missing_risk_ids": [
    "risk-1"
  ]`,
		`Do not output top-level "computed_summary"; runner computes it after validation`,
	} {
		if !strings.Contains(revisionPrompt, wantText) {
			t.Fatalf("revision prompt missing %q:\n%s", wantText, revisionPrompt)
		}
	}
}

func TestReviewRunnerRunRevisesReportWhenSaturationHasAdditionalCandidate(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	revisedReport := newPlanAwareHasFindingsReportForValidationTest()
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, needsRevisionAdditionalCandidateCheckForRunnerTest()))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, revisedReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got.Verdict != reviewreport.ReviewVerdictHasFindings {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, reviewreport.ReviewVerdictHasFindings)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseSaturationCheck,
	})
	if !strings.Contains(model.requests[3].Prompt, `"additional_finding_candidates"`) {
		t.Fatalf("revision prompt missing additional finding candidates:\n%s", model.requests[3].Prompt)
	}
}

func TestReviewRunnerRunDoesNotLoopWhenPostRevisionSaturationStillNeedsRevision(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	needsRevision := needsRevisionMissingRiskCheckForRunnerTest()
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, needsRevision))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newPlanAwareHasFindingsReportForValidationTest()))},
			{content: string(mustMarshalReviewSaturationCheckForTest(t, needsRevision))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newPlanAwareHasFindingsReportForValidationTest()))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want post-revision saturation error")
	}
	if !strings.Contains(err.Error(), "still needs revision after one revision") {
		t.Fatalf("Run() error = %q, want no-loop saturation error", err.Error())
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseSaturationCheck,
	})
}
