package review

import (
	"context"
	"reflect"
	"strings"
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
	if got.Verdict != ReviewVerdictHasFindings {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, ReviewVerdictHasFindings)
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
	if got.Verdict != ReviewVerdictClean {
		t.Fatalf("Run() verdict = %q, want %q", got.Verdict, ReviewVerdictClean)
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

func TestReviewRunnerRunBlockedSaturationReturnsError(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	blockedCheck := ReviewSaturationCheck{
		SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
		Status:         ReviewSaturationStatusBlocked,
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

func saturatedRunnerModelResponseForTest(t *testing.T) runnerFakeModelResponse {
	t.Helper()

	return runnerFakeModelResponse{
		content: string(mustMarshalReviewSaturationCheckForTest(t, newSaturatedReviewSaturationCheckForTest())),
	}
}

func needsRevisionMissingRiskCheckForRunnerTest() ReviewSaturationCheck {
	return ReviewSaturationCheck{
		SchemaVersion:        ReviewSaturationCheckSchemaVersionV1,
		Status:               ReviewSaturationStatusNeedsRevision,
		CheckedSummary:       "risk-1 was not fully represented in the finalized report.",
		MissingRiskIDs:       []string{"risk-1"},
		RevisionInstructions: "Revise the report so risk-1 is explicitly classified in scope_coverage.",
	}
}

func needsRevisionAdditionalCandidateCheckForRunnerTest() ReviewSaturationCheck {
	return ReviewSaturationCheck{
		SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
		Status:         ReviewSaturationStatusNeedsRevision,
		CheckedSummary: "A file-backed candidate was not represented in the finalized report.",
		AdditionalFindingCandidates: []ReviewSaturationAdditionalFindingCandidate{
			{
				Summary: "A report-pass finding candidate is grounded in existing file evidence.",
				EvidenceRefs: []ReviewEvidenceRef{
					newFileEvidenceRefForValidationTest(),
				},
				Reason: "The candidate uses existing evidence only and does not require additional exploration.",
			},
		},
		RevisionInstructions: "Revise the report to include or explicitly dismiss the file-backed candidate.",
	}
}

func newRunnerEvidenceBundleWithWebSearchForSaturationAuditTest(repoRoot string) ReviewEvidenceBundle {
	bundle := newRunnerEvidenceBundleForTest(repoRoot)
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:      true,
		Inconclusive: true,
	}
	return bundle
}

func newRunnerEvidenceBundleWithAdequateWebSearchSupportForSaturationAuditTest(repoRoot string) ReviewEvidenceBundle {
	bundle := newRunnerEvidenceBundleForTest(repoRoot)
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled: true,
		ExternalDocs: []ReviewExternalDocEvidence{
			newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
				"external-doc-official-1",
				"https://docs.example.test/oauth",
				"OAuth redirect URI official behavior.",
				"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			),
			newRunnerOfficialCandidateExternalDocForSaturationAuditTest(
				"external-doc-official-2",
				"https://reference.example.test/oauth",
				"OAuth redirect URI reference behavior.",
				"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			),
		},
	}
	return bundle
}

func newRunnerPostPass1WeakExternalEvidenceForSaturationAuditTest() ReviewWebSearchEvidence {
	return ReviewWebSearchEvidence{
		Enabled: true,
		Queries: []ReviewWebSearchEvidenceQuery{
			{
				Query:  "OAuth 2.0 redirect URI specification",
				Reason: "intent=spec; expected_source_type=technical_specification; confidence=high; reason=pass1 plan protocol/spec signal",
				Results: []ReviewWebSearchEvidenceResult{
					{Title: "OAuth 2.0 redirect URI specification", URL: "https://docs.example.test/oauth"},
				},
			},
		},
		ExternalDocs: []ReviewExternalDocEvidence{
			{
				DocID:             "external-doc-post",
				URL:               "https://docs.example.test/oauth",
				SourceCredibility: ReviewExternalDocSourceCredibilityUnknown,
				Snippets: []ReviewExternalDocSnippetEvidence{
					{
						SnippetID:   "external-doc-post-snippet-1",
						Content:     "Post-pass1 OAuth redirect URI snippet.",
						ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
					},
				},
			},
		},
	}
}
