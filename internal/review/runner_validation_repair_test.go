package review

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestReviewRunnerRunRepairsInvalidProbePlanJSON(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: `{not-json`},
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got.Verdict != ReviewVerdictClean {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, ReviewVerdictClean)
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
	for _, want := range []string{"Probe Plan JSON Repair", "Return corrected JSON only.", "{not-json", "decode review probe plan"} {
		if !strings.Contains(model.requests[1].Prompt, want) {
			t.Fatalf("probe plan repair prompt missing %q:\n%s", want, model.requests[1].Prompt)
		}
	}
}

func TestReviewRunnerRunRepairsInvalidProbePlanValidation(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanWithMissingNoProbeReasonForRunnerTest(t))},
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
	for _, want := range []string{"Probe Plan JSON Repair", "no_probe_reason must be non-empty"} {
		if !strings.Contains(model.requests[1].Prompt, want) {
			t.Fatalf("probe plan repair prompt missing %q:\n%s", want, model.requests[1].Prompt)
		}
	}
}

func TestReviewRunnerRunRepairsProbePlanEvidenceValidation(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	invalidPlan := newRunnerProbePlanForTest("probe-1")
	invalidPlan.ImpactSurfaces[0].EvidenceSummary = "Evidence references current review changes."
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, invalidPlan))},
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
	for _, want := range []string{"Probe Plan JSON Repair", "ValidateReviewProbePlanAgainstEvidence", "internal/review/runner.go"} {
		if !strings.Contains(model.requests[1].Prompt, want) {
			t.Fatalf("probe plan repair prompt missing %q:\n%s", want, model.requests[1].Prompt)
		}
	}
}

func TestReviewRunnerRunRepairsInvalidReportJSON(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: `{not-json`},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got.Verdict != ReviewVerdictClean {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, ReviewVerdictClean)
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
	for _, want := range []string{"Report JSON Repair", "Return corrected JSON only.", "Preserve trusted probe summary IDs", "{not-json", "decode review report"} {
		if !strings.Contains(model.requests[2].Prompt, want) {
			t.Fatalf("report repair prompt missing %q:\n%s", want, model.requests[2].Prompt)
		}
	}
}

func TestReviewRunnerRunRepairsInvalidReportValidation(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: `{"schema_version":"review_report.v2","target_kind":"current_changes"}`},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got.Verdict != ReviewVerdictClean {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, ReviewVerdictClean)
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
	for _, want := range []string{"Report JSON Repair", "finalize report", "generated_at must be non-zero"} {
		if !strings.Contains(model.requests[2].Prompt, want) {
			t.Fatalf("report repair prompt missing %q:\n%s", want, model.requests[2].Prompt)
		}
	}
}

func TestReviewRunnerRunRepairsReportScopeCoverageValidation(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	invalidReport := newRunnerCleanReportForTest(nil)
	invalidReport.ScopeCoverage = nil
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, invalidReport))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got.Verdict != ReviewVerdictClean {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, ReviewVerdictClean)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseReport,
		ReviewModelPhaseSaturationCheck,
	})
	for _, want := range []string{"Report JSON Repair", "finalize report", "scope_coverage is required"} {
		if !strings.Contains(model.requests[2].Prompt, want) {
			t.Fatalf("report repair prompt missing %q:\n%s", want, model.requests[2].Prompt)
		}
	}
}

func TestReviewRunnerRunPass1RepairModelErrorUsesPass1ModelContract(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: `{not-json`},
			{err: errors.New("repair model unavailable")},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	for _, want := range []string{"pass1 model", "repair model unavailable"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Run() error = %q, want substring %q", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "pass1 repair model") {
		t.Fatalf("Run() error = %q, should not introduce a repair model error prefix", err.Error())
	}
	if got, want := len(probes.calls), 0; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseProbePlan,
	})
}

func TestReviewRunnerRunPass2RepairModelErrorUsesPass2ModelContract(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: `{not-json`},
			{err: errors.New("repair model unavailable")},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	for _, want := range []string{"pass2 model", "repair model unavailable"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Run() error = %q, want substring %q", err.Error(), want)
		}
	}
	if strings.Contains(err.Error(), "pass2 repair model") {
		t.Fatalf("Run() error = %q, should not introduce a repair model error prefix", err.Error())
	}
	if got, want := len(probes.calls), 1; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	assertReviewRunnerRequestPhasesForTest(t, model.requests, []ReviewModelPhase{
		ReviewModelPhaseProbePlan,
		ReviewModelPhaseReport,
		ReviewModelPhaseReport,
	})
}
