package review

import (
	"context"
	"reflect"
	"strings"
	"testing"
)

func TestReviewRunnerRunRevisesReportWhenCoverageAuditFindsUnsupportedOfficialConfirmation(t *testing.T) {
	evidence := &runnerPostPass1WebSearchEvidenceBuilder{
		runnerFakeEvidenceBuilder: runnerFakeEvidenceBuilder{
			bundle: newRunnerEvidenceBundleWithWebSearchForSaturationAuditTest("/tmp/review-runner/repo"),
		},
		postEvidence: newRunnerPostPass1WeakExternalEvidenceForSaturationAuditTest(),
	}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	initialReport := newRunnerCleanReportForTest(nil)
	initialReport.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked; official documentation confirms this behavior."
	initialReport.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed; confirmed external spec coverage applies."
	revisedReport := newRunnerCleanReportForTest(nil)
	revisedReport.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked after reviewing weak external evidence external-doc-post; official confirmation is absent."
	revisedReport.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed after reviewing weak external evidence external-doc-post; official confirmation is absent."
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, initialReport))},
			saturatedRunnerModelResponseForTest(t),
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
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseSaturationCheck,
	})
	revisionPrompt := model.requests[3].Prompt
	for _, wantText := range []string{
		"Deterministic coverage audit requires revision",
		"unsupported_external_confirmation",
		"Remove or qualify official confirmation",
	} {
		if !strings.Contains(revisionPrompt, wantText) {
			t.Fatalf("revision prompt missing coverage audit feedback %q:\n%s", wantText, revisionPrompt)
		}
	}
}

func TestMergeReviewCoverageAuditIntoSaturationCheckRevisesIgnoredNonPassingProbe(t *testing.T) {
	plan := newRunnerProbePlanForTest("probe-1")
	probeSummary := ReviewProbeSummary{
		ProbeID: "probe-1",
		Mode:    ReviewProbeHostReadOnly,
		Status:  ReviewProbeFailed,
		Error:   "go test ./internal/review failed",
	}
	report := newRunnerCleanReportForTest([]ReviewProbeSummary{probeSummary})

	merged, err := mergeReviewCoverageAuditIntoSaturationCheck(
		newSaturatedReviewSaturationCheckForTest(),
		plan,
		report,
		[]ReviewProbeSummary{probeSummary},
		reviewCoverageAuditContext{},
	)
	if err != nil {
		t.Fatalf("mergeReviewCoverageAuditIntoSaturationCheck() error = %v, want nil", err)
	}
	if merged.Status != ReviewSaturationStatusNeedsRevision {
		t.Fatalf("Status = %q, want %q", merged.Status, ReviewSaturationStatusNeedsRevision)
	}
	if len(merged.AdditionalFindingCandidates) == 0 {
		t.Fatalf("AdditionalFindingCandidates = %#v, want probe-backed candidate", merged.AdditionalFindingCandidates)
	}
	for _, want := range []string{
		"unreflected_probe_outcome",
		"Revisit the scope linked to non-passing probe",
		"revise the report before deciding whether this remains a finding candidate",
	} {
		if !strings.Contains(merged.RevisionInstructions+merged.AdditionalFindingCandidates[0].Reason, want) {
			t.Fatalf("merged feedback missing %q: %#v", want, merged)
		}
	}
}

func TestReviewRunnerRunStopsWhenCoverageAuditStillNeedsRevisionAfterOneRevision(t *testing.T) {
	evidence := &runnerPostPass1WebSearchEvidenceBuilder{
		runnerFakeEvidenceBuilder: runnerFakeEvidenceBuilder{
			bundle: newRunnerEvidenceBundleWithWebSearchForSaturationAuditTest("/tmp/review-runner/repo"),
		},
		postEvidence: newRunnerPostPass1WeakExternalEvidenceForSaturationAuditTest(),
	}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	unsupportedReport := newRunnerCleanReportForTest(nil)
	unsupportedReport.ScopeCoverage.ReviewedImpactSurfaces[0].Summary = "surface-1 checked; official documentation confirms this behavior."
	unsupportedReport.ScopeCoverage.ReviewedCandidateRisks[0].Summary = "risk-1 dismissed; confirmed external spec coverage applies."
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, unsupportedReport))},
			saturatedRunnerModelResponseForTest(t),
			{content: string(mustMarshalReviewReportForRunnerTest(t, unsupportedReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want post-revision coverage audit error")
	}
	if !strings.Contains(err.Error(), "still needs revision after one revision") ||
		!strings.Contains(err.Error(), "unsupported_external_confirmation") {
		t.Fatalf("Run() error = %q, want coverage audit no-loop error", err.Error())
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
		ReviewModelPhaseReportRevision,
		ReviewModelPhaseSaturationCheck,
	})
}
