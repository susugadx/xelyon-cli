package review

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestReviewRunnerRunHappyPath(t *testing.T) {
	events := []string{}
	evidence := &runnerFakeEvidenceBuilder{
		bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo"),
		events: &events,
	}
	probeResult := ReviewProbeResult{
		ID:              "probe-1",
		Mode:            ReviewProbeHostReadOnly,
		Status:          ReviewProbeFailed,
		OutputTruncated: true,
		Error:           "probe failed",
		CommandResults: []ReviewProbeCommandResult{
			{
				Command:         "go",
				Args:            []string{"test", "./internal/review"},
				Status:          ReviewProbeFailed,
				ExitCode:        1,
				Output:          "FAIL runner",
				OutputTruncated: true,
				Error:           "exit status 1",
				Duration:        1500 * time.Millisecond,
			},
		},
	}
	probes := &runnerFakeProbeRunner{
		results: map[string]ReviewProbeResult{"probe-1": probeResult},
		events:  &events,
	}
	plan := newRunnerProbePlanForTest("probe-1")
	probeResults := []ReviewProbeResult{probeResult}
	report := newRunnerCleanReportForTest(newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, probeResults))
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
		},
		events: &events,
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest("focus on runner orchestration"))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}

	if !reflect.DeepEqual(got, report) {
		t.Fatalf("Run() report = %#v, want %#v", got, report)
	}
	assertStringSliceEqualForRunnerTest(t, events, []string{
		"evidence",
		"model:probe_plan",
		"probe:probe-1",
		"model:report",
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
	if got, want := len(model.requests), 2; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
	if got, want := model.requests[0].Phase, ReviewModelPhaseProbePlan; got != want {
		t.Fatalf("first model phase = %q, want %q", got, want)
	}
	if got, want := model.requests[1].Phase, ReviewModelPhaseReport; got != want {
		t.Fatalf("second model phase = %q, want %q", got, want)
	}
	firstPrompt := model.requests[0].Prompt
	for _, want := range []string{"Review Pass 1", "focus on runner orchestration", "# Review Evidence", ReviewProbePlanSchemaVersionV2} {
		if !strings.Contains(firstPrompt, want) {
			t.Fatalf("Pass1 prompt missing %q:\n%s", want, firstPrompt)
		}
	}
	secondPrompt := model.requests[1].Prompt
	for _, want := range []string{
		"Review Pass 2",
		ReviewReportSchemaVersionV2,
		"Probe Result Context",
		`"output": "FAIL runner"`,
		`"status": "failed"`,
		`"error": "exit status 1"`,
		`"output_truncated": true`,
		`"duration_ms": 1500`,
	} {
		if !strings.Contains(secondPrompt, want) {
			t.Fatalf("Pass2 prompt missing %q:\n%s", want, secondPrompt)
		}
	}
}

func TestReviewRunnerRunUsesTrustedProbeSummaries(t *testing.T) {
	repoRoot := t.TempDir()
	probeRoot := filepath.Join(t.TempDir(), reviewProbeSandboxTempPrefix+"trusted")
	probeWorkDir := filepath.Join(probeRoot, "worktree")
	repoFile := filepath.Join(repoRoot, "internal/review/runner.go")
	probeFile := filepath.Join(probeWorkDir, "output.txt")
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest(repoRoot)}
	probeResult := ReviewProbeResult{
		ID:              "probe-1",
		Mode:            ReviewProbeHostReadOnly,
		Status:          ReviewProbeFailed,
		MutatedFiles:    []string{repoFile, probeFile},
		OutputTruncated: true,
		Error:           "probe failed at " + repoFile + " and " + probeFile,
		CommandResults: []ReviewProbeCommandResult{
			{
				Command:         "cat " + probeFile,
				Args:            []string{repoFile, probeFile},
				WorkDir:         probeWorkDir,
				Status:          ReviewProbeFailed,
				ExitCode:        1,
				OutputTruncated: true,
				Error:           "exit status 1 at " + probeFile,
				Duration:        1500 * time.Millisecond,
			},
		},
	}
	modelReport := newRunnerCleanReportForTest([]ReviewProbeSummary{
		{
			ProbeID:      "fake-probe",
			Mode:         ReviewProbeHostReadOnly,
			Status:       ReviewProbePassed,
			MutatedFiles: []string{"/fake/model/path"},
			Error:        "fake model summary must be ignored",
		},
	})
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	wantSummaries := []ReviewProbeSummary{
		{
			ProbeID:         "probe-1",
			Mode:            ReviewProbeHostReadOnly,
			Status:          ReviewProbeFailed,
			MutatedFiles:    []string{"internal/review/runner.go", "<probe_workdir>/output.txt"},
			OutputTruncated: true,
			Error:           "probe failed at <repo_root>/internal/review/runner.go and <probe_workdir>/output.txt",
			Commands: []ReviewProbeCommandSummary{
				{
					Command:         "cat <probe_workdir>/output.txt",
					Args:            []string{"<repo_root>/internal/review/runner.go", "<probe_workdir>/output.txt"},
					WorkDir:         "<probe_workdir>",
					Status:          ReviewProbeFailed,
					ExitCode:        1,
					OutputTruncated: true,
					Error:           "exit status 1 at <probe_workdir>/output.txt",
					DurationMs:      1500,
				},
			},
		},
	}
	if !reflect.DeepEqual(got.ProbeSummaries, wantSummaries) {
		t.Fatalf("Run() probe summaries = %#v, want redacted trusted %#v", got.ProbeSummaries, wantSummaries)
	}
	assertReviewReportDoesNotContainForRunnerTest(t, got, repoRoot, probeRoot, "/fake/model/path", "fake model summary")
}

func TestReviewRunnerRunInjectsTrustedProbeSummariesBeforeReportValidation(t *testing.T) {
	repoRoot := t.TempDir()
	repoFile := filepath.Join(repoRoot, "internal/review/runner.go")
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest(repoRoot)}
	probeResult := ReviewProbeResult{
		ID:     "probe-1",
		Mode:   ReviewProbeHostReadOnly,
		Status: ReviewProbePassed,
		CommandResults: []ReviewProbeCommandResult{
			{
				Command: "cat " + repoFile,
				Args:    []string{repoFile},
				WorkDir: repoRoot,
				Status:  ReviewProbePassed,
			},
		},
	}
	modelReport := newRunnerCleanReportForTest(nil)
	modelReport.CheckedSurfaces = []ReviewSurfaceCoverage{
		{
			SurfaceID: "surface-1",
			EvidenceRefs: []ReviewEvidenceRef{
				{
					Kind:         ReviewEvidenceKindProbeCommand,
					ProbeID:      "probe-1",
					CommandIndex: ReviewCommandIndex(0),
				},
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	wantSummaries := []ReviewProbeSummary{
		{
			ProbeID:      "probe-1",
			Mode:         ReviewProbeHostReadOnly,
			Status:       ReviewProbePassed,
			MutatedFiles: []string{},
			Commands: []ReviewProbeCommandSummary{
				{
					Command: "cat <repo_root>/internal/review/runner.go",
					Args:    []string{"<repo_root>/internal/review/runner.go"},
					WorkDir: ".",
					Status:  ReviewProbePassed,
				},
			},
		},
	}
	if !reflect.DeepEqual(got.ProbeSummaries, wantSummaries) {
		t.Fatalf("Run() probe summaries = %#v, want redacted trusted probe summaries %#v", got.ProbeSummaries, wantSummaries)
	}
	if got.CheckedSurfaces[0].EvidenceRefs[0].ProbeID != "probe-1" {
		t.Fatalf("EvidenceRef probe_id = %q, want probe-1", got.CheckedSurfaces[0].EvidenceRefs[0].ProbeID)
	}
	if got.CheckedSurfaces[0].EvidenceRefs[0].CommandIndex == nil || *got.CheckedSurfaces[0].EvidenceRefs[0].CommandIndex != 0 {
		t.Fatalf("EvidenceRef command_index = %#v, want 0", got.CheckedSurfaces[0].EvidenceRefs[0].CommandIndex)
	}
	assertReviewReportDoesNotContainForRunnerTest(t, got, repoRoot)
}

func TestReviewRunnerRunRevalidatesReportAfterTrustedProbeSummaries(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probeResult := ReviewProbeResult{
		ID:     "probe-1",
		Mode:   ReviewProbeHostReadOnly,
		Status: ReviewProbePassed,
		CommandResults: []ReviewProbeCommandResult{
			{
				Command: "go",
				Status:  ReviewProbePassed,
			},
		},
	}
	modelReport := newRunnerCleanReportForTest([]ReviewProbeSummary{
		{
			ProbeID: "fake-probe",
			Mode:    ReviewProbeHostReadOnly,
			Status:  ReviewProbePassed,
			Commands: []ReviewProbeCommandSummary{
				{
					Command: "go",
					Status:  ReviewProbePassed,
				},
			},
		},
	})
	modelReport.CheckedSurfaces = []ReviewSurfaceCoverage{
		{
			SurfaceID: "surface-1",
			EvidenceRefs: []ReviewEvidenceRef{
				{
					Kind:         ReviewEvidenceKindProbeCommand,
					ProbeID:      "fake-probe",
					CommandIndex: ReviewCommandIndex(0),
				},
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	for _, want := range []string{"finalize report", "unknown probe_id"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Run() error = %q, want substring %q", err.Error(), want)
		}
	}
	if got, want := len(model.requests), 3; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
}

func TestReviewRunnerRunNoProbeReasonSkipsProbeRunner(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	report := newRunnerCleanReportForTest([]ReviewProbeSummary{})
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(got, report) {
		t.Fatalf("Run() report = %#v, want %#v", got, report)
	}
	if got, want := len(probes.calls), 0; got != want {
		t.Fatalf("probe calls = %d, want %d", got, want)
	}
	if got, want := len(model.requests), 2; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
	if !strings.Contains(model.requests[1].Prompt, `"no_probe_reason": "surface-1 and risk-1 are checked by existing evidence."`) {
		t.Fatalf("Pass2 prompt missing no_probe_reason:\n%s", model.requests[1].Prompt)
	}
}

func TestReviewRunnerRunRepairsInvalidProbePlanJSON(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: `{not-json`},
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
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
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
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
	})
	for _, want := range []string{"Probe Plan JSON Repair", "no_probe_reason must be non-empty"} {
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
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
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
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
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
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
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
	})
	for _, want := range []string{"Report JSON Repair", "finalize report", "scope_coverage is required"} {
		if !strings.Contains(model.requests[2].Prompt, want) {
			t.Fatalf("report repair prompt missing %q:\n%s", want, model.requests[2].Prompt)
		}
	}
}

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

func TestNewReviewRunnerRejectsNilDependencies(t *testing.T) {
	valid := newRunnerNonNilDependenciesForTest()
	tests := []struct {
		name        string
		opts        ReviewRunnerOptions
		errContains string
	}{
		{
			name:        "model",
			opts:        ReviewRunnerOptions{EvidenceBuilder: valid.EvidenceBuilder, ProbeRunner: valid.ProbeRunner},
			errContains: "review runner model is nil",
		},
		{
			name:        "evidence builder",
			opts:        ReviewRunnerOptions{Model: valid.Model, ProbeRunner: valid.ProbeRunner},
			errContains: "review runner evidence builder is nil",
		},
		{
			name:        "probe runner",
			opts:        ReviewRunnerOptions{Model: valid.Model, EvidenceBuilder: valid.EvidenceBuilder},
			errContains: "review runner probe runner is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewReviewRunner(tt.opts)
			if err == nil {
				t.Fatal("NewReviewRunner() error = nil, want error")
			}
			if got := err.Error(); got != tt.errContains {
				t.Fatalf("NewReviewRunner() error = %q, want %q", got, tt.errContains)
			}
		})
	}
}

func TestReviewRunnerRunRejectsNilDependencies(t *testing.T) {
	valid := newRunnerNonNilDependenciesForTest()
	tests := []struct {
		name        string
		runner      *ReviewRunner
		errContains string
	}{
		{
			name: "model",
			runner: &ReviewRunner{
				evidenceBuilder: valid.EvidenceBuilder,
				probeRunner:     valid.ProbeRunner,
			},
			errContains: "review runner model is nil",
		},
		{
			name: "evidence builder",
			runner: &ReviewRunner{
				model:       valid.Model,
				probeRunner: valid.ProbeRunner,
			},
			errContains: "review runner evidence builder is nil",
		},
		{
			name: "probe runner",
			runner: &ReviewRunner{
				model:           valid.Model,
				evidenceBuilder: valid.EvidenceBuilder,
			},
			errContains: "review runner probe runner is nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.runner.Run(context.Background(), NewCurrentChangesRequest(""))
			if err == nil {
				t.Fatal("Run() error = nil, want error")
			}
			if got := err.Error(); got != tt.errContains {
				t.Fatalf("Run() error = %q, want %q", got, tt.errContains)
			}
		})
	}
}

func TestReviewRunnerRunRejectsUnknownTargetBeforeEvidenceBuild(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	model := &runnerFakeModel{}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), ReviewRequest{TargetKind: TargetKind("staged")})
	if err == nil {
		t.Fatal("Run() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "target_kind") {
		t.Fatalf("Run() error = %q, want target_kind", err.Error())
	}
	if got, want := evidence.calls, 0; got != want {
		t.Fatalf("evidence calls = %d, want %d", got, want)
	}
}

func TestReviewRunnerPromptRedactsAbsoluteRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest(repoRoot)}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	report := newRunnerCleanReportForTest([]ReviewProbeSummary{})
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, report))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	_, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, want := len(model.requests), 2; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}
	for i, req := range model.requests {
		if strings.Contains(req.Prompt, repoRoot) {
			t.Fatalf("prompt %d contains absolute repo root %q:\n%s", i, repoRoot, req.Prompt)
		}
		if !strings.Contains(req.Prompt, "<repo_root>") {
			t.Fatalf("prompt %d missing <repo_root>:\n%s", i, req.Prompt)
		}
	}
}

func TestReviewRunnerRedactsProbeResultPathsInPass2PromptAndFinalReport(t *testing.T) {
	repoRoot := t.TempDir()
	probeRoot := filepath.Join(t.TempDir(), reviewProbeSandboxTempPrefix+"abc")
	probeWorkDir := filepath.Join(probeRoot, "worktree")
	scratchRoot := filepath.Join(t.TempDir(), reviewProbeScratchTempPrefix+"def")
	repoFile := filepath.Join(repoRoot, "internal/review/runner.go")
	probeWorkFile := filepath.Join(probeWorkDir, "output.txt")
	probeRuntimeFile := filepath.Join(probeRoot, "runtime/home/output.txt")
	scratchFile := filepath.Join(scratchRoot, "tmp/mutated.txt")
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest(repoRoot)}
	probeResult := ReviewProbeResult{
		ID:              "probe-1",
		Mode:            ReviewProbeHostReadOnly,
		Status:          ReviewProbeFailed,
		MutatedWorktree: true,
		MutatedFiles:    []string{repoFile, filepath.Join(probeWorkDir, "mutated.txt"), scratchFile},
		Error:           "probe failed at " + repoFile + " using " + probeRuntimeFile,
		CommandResults: []ReviewProbeCommandResult{
			{
				Command: "pwd",
				WorkDir: repoRoot,
				Status:  ReviewProbeFailed,
				Output:  "repo path: " + repoFile,
				Error:   "repo error: " + repoRoot,
			},
			{
				Command: "cat",
				Args:    []string{probeWorkFile, scratchFile},
				WorkDir: probeWorkDir,
				Status:  ReviewProbeFailed,
				Output:  "probe path: " + probeWorkFile,
				Error:   "probe runtime error: " + probeRuntimeFile,
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(nil)))},
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, want := len(model.requests), 2; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}

	secondPrompt := model.requests[1].Prompt
	for _, leaked := range []string{repoRoot, repoFile, probeRoot, probeWorkDir, probeWorkFile, probeRuntimeFile, scratchRoot, scratchFile} {
		if strings.Contains(secondPrompt, leaked) {
			t.Fatalf("Pass2 prompt contains absolute path %q:\n%s", leaked, secondPrompt)
		}
	}
	for _, want := range []string{
		"<repo_root>/internal/review/runner.go",
		"<probe_workdir>/output.txt",
		"<probe_workdir>/runtime/home/output.txt",
		`"work_dir": "."`,
		`"work_dir": "<probe_workdir>"`,
	} {
		if !strings.Contains(secondPrompt, want) {
			t.Fatalf("Pass2 prompt missing %q:\n%s", want, secondPrompt)
		}
	}

	assertReviewReportDoesNotContainForRunnerTest(t, got, repoRoot, repoFile, probeRoot, probeWorkDir, probeWorkFile, probeRuntimeFile, scratchRoot, scratchFile)
	if got, want := got.ProbeSummaries[0].MutatedFiles, []string{
		"internal/review/runner.go",
		"<probe_workdir>/mutated.txt",
		"<probe_workdir_2>/tmp/mutated.txt",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("final report MutatedFiles = %#v, want %#v", got, want)
	}
	if got, want := got.ProbeSummaries[0].Status, ReviewProbeMutatedWorktree; got != want {
		t.Fatalf("final report probe status = %q, want %q", got, want)
	}
	if !got.ProbeSummaries[0].MutatedWorktree {
		t.Fatal("final report MutatedWorktree = false, want true")
	}
	if got, want := got.ProbeSummaries[0].Error, "probe failed at <repo_root>/internal/review/runner.go using <probe_workdir>/runtime/home/output.txt"; got != want {
		t.Fatalf("final report probe error = %q, want %q", got, want)
	}
	if got, want := got.ProbeSummaries[0].Commands[0].WorkDir, "."; got != want {
		t.Fatalf("final report command 0 WorkDir = %q, want %q", got, want)
	}
	if got, want := got.ProbeSummaries[0].Commands[1].WorkDir, "<probe_workdir>"; got != want {
		t.Fatalf("final report command 1 WorkDir = %q, want %q", got, want)
	}
	if got, want := got.ProbeSummaries[0].Commands[1].Args, []string{"<probe_workdir>/output.txt", "<probe_workdir_2>/tmp/mutated.txt"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("final report command 1 Args = %#v, want %#v", got, want)
	}
}

func TestReviewRunnerRunsProbesInPlanOrder(t *testing.T) {
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerProbePlanForTest("probe-a", "probe-b", "probe-c")
	results := []ReviewProbeResult{
		{ID: "probe-a", Mode: ReviewProbeHostReadOnly, Status: ReviewProbePassed},
		{ID: "probe-b", Mode: ReviewProbeHostReadOnly, Status: ReviewProbePassed},
		{ID: "probe-c", Mode: ReviewProbeHostReadOnly, Status: ReviewProbePassed},
	}
	probes.results = map[string]ReviewProbeResult{
		"probe-a": results[0],
		"probe-b": results[1],
		"probe-c": results[2],
	}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerCleanReportForTest(BuildReviewProbeSummaries(results))))},
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

type runnerFakeEvidenceBuilder struct {
	bundle ReviewEvidenceBundle
	err    error
	calls  int
	events *[]string
}

func (b *runnerFakeEvidenceBuilder) BuildCurrentChanges(context.Context) (ReviewEvidenceBundle, error) {
	b.calls++
	if b.events != nil {
		*b.events = append(*b.events, "evidence")
	}
	if b.err != nil {
		return ReviewEvidenceBundle{}, b.err
	}
	return b.bundle, nil
}

type runnerFakeProbeRunner struct {
	results map[string]ReviewProbeResult
	errors  map[string]error
	calls   []ReviewProbeRequest
	events  *[]string
}

func (r *runnerFakeProbeRunner) Run(_ context.Context, req ReviewProbeRequest) (ReviewProbeResult, error) {
	r.calls = append(r.calls, req)
	if r.events != nil {
		*r.events = append(*r.events, "probe:"+req.ID)
	}
	if err := r.errors[req.ID]; err != nil {
		return ReviewProbeResult{}, err
	}
	result, ok := r.results[req.ID]
	if !ok {
		return ReviewProbeResult{
			ID:     req.ID,
			Mode:   req.Mode,
			Status: ReviewProbePassed,
		}, nil
	}
	if result.ID == "" {
		result.ID = req.ID
	}
	if result.Mode == "" {
		result.Mode = req.Mode
	}
	if result.Status == "" {
		result.Status = ReviewProbePassed
	}
	return result, nil
}

type runnerFakeModel struct {
	responses []runnerFakeModelResponse
	requests  []ReviewModelRequest
	events    *[]string
}

type runnerFakeModelResponse struct {
	content string
	err     error
}

func (m *runnerFakeModel) CompleteReview(_ context.Context, req ReviewModelRequest) (ReviewModelResponse, error) {
	m.requests = append(m.requests, req)
	if m.events != nil {
		*m.events = append(*m.events, "model:"+string(req.Phase))
	}
	index := len(m.requests) - 1
	if index >= len(m.responses) {
		return ReviewModelResponse{}, errors.New("unexpected review model call")
	}
	response := m.responses[index]
	if response.err != nil {
		return ReviewModelResponse{}, response.err
	}
	return ReviewModelResponse{Content: response.content}, nil
}

func newReviewRunnerForTest(t *testing.T, evidence ReviewEvidenceProvider, probes ReviewProbeExecutor, model ReviewModel) *ReviewRunner {
	t.Helper()

	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder: evidence,
		ProbeRunner:     probes,
		Model:           model,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}
	return runner
}

func newRunnerNonNilDependenciesForTest() ReviewRunnerOptions {
	return ReviewRunnerOptions{
		EvidenceBuilder: &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")},
		ProbeRunner:     &runnerFakeProbeRunner{},
		Model:           &runnerFakeModel{},
	}
}

func newRunnerEvidenceBundleForTest(repoRoot string) ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		TargetKind:  TargetCurrentChanges,
		RepoRoot:    repoRoot,
		CWD:         filepath.Join(repoRoot, "internal"),
		StatusShort: " M internal/review/runner.go\n",
		ChangedFiles: []ReviewChangedFile{
			{
				Path:     filepath.Join(repoRoot, "internal/review/runner.go"),
				Status:   "M",
				Unstaged: true,
			},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{filepath.Join(repoRoot, "internal/review/runner.go")},
		},
		Limits: DefaultReviewEvidenceLimits(),
	}
}

func newRunnerProbePlanForTest(ids ...string) ReviewProbePlan {
	probes := make([]ReviewPlannedProbe, 0, len(ids))
	for _, id := range ids {
		probes = append(probes, ReviewPlannedProbe{
			ID:         id,
			SurfaceIDs: []string{"surface-1"},
			RiskIDs:    []string{"risk-1"},
			Purpose:    "Confirm or falsify risk-1 for surface-1 with focused review checks.",
			Mode:       ReviewProbeHostReadOnly,
			Commands: []ReviewPlannedProbeCommand{
				{
					Command: "go",
					Args:    []string{"test", "./internal/review"},
					WorkDir: ".",
				},
			},
			TimeoutSeconds: 30,
			MaxOutputBytes: 4096,
		})
	}
	return ReviewProbePlan{
		SchemaVersion: ReviewProbePlanSchemaVersionV2,
		TargetKind:    TargetCurrentChanges,
		Summary:       "Probe current changes.",
		ImpactSurfaces: []ReviewProbeImpactSurface{
			{
				ID:              "surface-1",
				Summary:         "Runner orchestration may need verification.",
				Category:        ReviewProbeImpactSurfaceChangedFile,
				EvidenceSummary: "Evidence references current review changes.",
				Status:          ReviewProbeImpactSurfaceNeedsProbe,
				Reason:          "Run the planned probes in order.",
			},
		},
		CandidateRisks: []ReviewProbeCandidateRisk{
			{
				ID:                   "risk-1",
				Summary:              "A runner contract could regress.",
				Severity:             ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-1"},
				EvidenceSummary:      "Runner tests cover probe orchestration.",
				VerificationStrategy: "Execute the focused runner probe.",
				Status:               ReviewProbeCandidateRiskNeedsProbe,
			},
		},
		Probes: probes,
	}
}

func newRunnerNoProbePlanForTest() ReviewProbePlan {
	return markReviewProbePlanCheckedWithoutProbesForTest(newRunnerProbePlanForTest())
}

func newRunnerCleanReportForTest(probeSummaries []ReviewProbeSummary) ReviewReport {
	var reportProbeSummaries []ReviewProbeSummary
	if len(probeSummaries) > 0 {
		reportProbeSummaries = probeSummaries
	}
	return ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV2,
		TargetKind:                TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: ReviewVerificationVerified,
		Verdict:                   ReviewVerdictClean,
		ProbeSummaries:            reportProbeSummaries,
		ScopeCoverage:             newCleanScopeCoverageForTest(),
	}
}

func newRunnerBlockedReportForTest(probeSummaries []ReviewProbeSummary) ReviewReport {
	report := newRunnerCleanReportForTest(probeSummaries)
	report.OverallVerificationStatus = ReviewVerificationBlockedOrInconclusive
	report.Verdict = ReviewVerdictBlocked
	report.Summary = "Review blocked by probe execution."
	return report
}

func mustMarshalReviewProbePlanForRunnerTest(t *testing.T, plan ReviewProbePlan) []byte {
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

func mustMarshalReviewReportForRunnerTest(t *testing.T, report ReviewReport) []byte {
	t.Helper()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func newRedactedRunnerProbeSummariesForTest(t *testing.T, bundle ReviewEvidenceBundle, results []ReviewProbeResult) []ReviewProbeSummary {
	t.Helper()

	return redactReviewProbeSummaries(
		BuildReviewProbeSummaries(results),
		newReviewRunnerPromptRedactor(bundle, results),
	)
}

func assertReviewReportDoesNotContainForRunnerTest(t *testing.T, report ReviewReport, leakedValues ...string) {
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
