package review

import (
	"context"
	"reflect"
	"testing"
)

func TestReviewRunnerRunReturnsInitialReportWhenSaturationIsSatisfied(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	report := newRunnerCleanReportForTest(nil)
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	want := withComputedSummaryForRunnerTest(report, nil)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Run() report = %#v, want %#v", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
}

func TestReviewRunnerRunReturnsInitialReportWhenWeakPostPassExternalEvidenceIsOnlyAdvisory(t *testing.T) {
	evidence := &runnerPostPass1WebSearchEvidenceBuilder{
		runnerFakeEvidenceBuilder: runnerFakeEvidenceBuilder{
			bundle: newRunnerEvidenceBundleWithWebSearchForSaturationAuditTest("/tmp/review-runner/repo"),
		},
		postEvidence: newRunnerPostPass1WeakExternalEvidenceForSaturationAuditTest(),
	}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	initialReport := newRunnerCleanReportForTest(nil)
	initialReport.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked for OAuth redirect URI validation."
	initialReport.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed for OAuth redirect URI validation."
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, initialReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	want := withComputedSummaryForRunnerTest(initialReport, nil)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Run() report = %#v, want initial %#v", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
}

func TestReviewRunnerRunPreservesExternalSupportWithoutPostPass1Provider(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{
		bundle: newRunnerEvidenceBundleWithAdequateWebSearchSupportForSaturationAuditTest("/tmp/review-runner/repo"),
	}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	report := newRunnerCleanReportForTest(nil)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked; official documentation confirms this behavior."
	report.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed; confirmed external spec coverage applies."
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	want := withComputedSummaryForRunnerTest(report, nil)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Run() report = %#v, want %#v", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
}
