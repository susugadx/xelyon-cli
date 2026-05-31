package review

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewRunnerRunIgnoresArtifactWriteFailureAndWarns(t *testing.T) {
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
	var warnings bytes.Buffer
	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:       evidence,
		ProbeRunner:           probes,
		Model:                 model,
		ArtifactWriter:        failingReviewRunArtifactWriter{err: errors.New("disk full")},
		ArtifactWarningWriter: &warnings,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v", err)
	}

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil despite artifact write failure", err)
	}
	if got.Verdict != ReviewVerdictClean {
		t.Fatalf("Run() verdict = %q, want clean", got.Verdict)
	}
	for _, want := range []string{
		"Warning: failed to save review artifact evidence.md: disk full",
		"Warning: failed to save review artifact report_final.json: disk full",
	} {
		if !strings.Contains(warnings.String(), want) {
			t.Fatalf("warnings missing %q:\n%s", want, warnings.String())
		}
	}
}

func TestReviewRunnerRunWithNilArtifactWriterDoesNotWarn(t *testing.T) {
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
	var warnings bytes.Buffer
	runner := newReviewRunnerWithArtifactWriterForTest(t, evidence, probes, model, nil, &warnings)

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if warnings.Len() != 0 {
		t.Fatalf("artifact warning output = %q, want empty when writer is nil", warnings.String())
	}
}

func TestReviewRunnerRunSavesHappyPathArtifacts(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v", err)
	}
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleWithWebSearchArtifactForTest("/tmp/review-runner/repo")}
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
	runner := newReviewRunnerWithArtifactWriterForTest(t, evidence, probes, model, writer, nil)

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("focus artifacts")); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertReviewArtifactsExistForTest(t, dir,
		"evidence.md",
		"web_search_evidence.json",
		"probe_plan_prompt.md",
		"probe_plan_raw.json",
		"probe_plan_final.json",
		"probe_requests.json",
		"probe_results.json",
		"report_prompt.md",
		"report_raw.json",
		"report_final.json",
		"saturation_prompt.md",
		"saturation_raw.json",
	)
}

func TestReviewRunnerRunSavesRevisionRepairArtifactsWithFinalReportSuffix(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v", err)
	}
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
	runner := newReviewRunnerWithArtifactWriterForTest(t, evidence, probes, model, writer, nil)

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("focus revision artifacts")); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	assertReviewArtifactsExistForTest(t, dir,
		"revision_prompt.md",
		"revision_prompt_2.md",
		"revision_raw.json",
		"revision_raw_2.json",
		"report_final.json",
		"report_final_2.json",
		"saturation_prompt.md",
		"saturation_prompt_2.md",
		"saturation_raw.json",
		"saturation_raw_2.json",
	)
}

func TestReviewRunnerRunRedactsProbePathsBeforeSavingArtifacts(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewReviewRunDirectoryArtifactWriter(dir)
	if err != nil {
		t.Fatalf("NewReviewRunDirectoryArtifactWriter() error = %v", err)
	}
	repoRoot := t.TempDir()
	probeRoot := filepath.Join(t.TempDir(), reviewProbeSandboxTempPrefix+"artifact")
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
		Error:           "failed at " + repoFile + " and " + probeFile,
		CommandResults: []ReviewProbeCommandResult{
			{
				Command:         "cat " + probeFile,
				Args:            []string{repoFile, probeFile},
				WorkDir:         probeWorkDir,
				Status:          ReviewProbeFailed,
				ExitCode:        1,
				Output:          "output from " + repoFile + " and " + probeFile,
				OutputTruncated: true,
				Error:           "exit at " + probeFile,
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerBlockedReportForTest(nil)))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerWithArtifactWriterForTest(t, evidence, probes, model, writer, nil)

	if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	artifact := string(readReviewArtifactForTest(t, filepath.Join(dir, "probe_results.json")))
	for _, leaked := range []string{repoRoot, repoFile, probeRoot, probeWorkDir, probeFile} {
		if strings.Contains(artifact, leaked) {
			t.Fatalf("probe_results.json leaked %q:\n%s", leaked, artifact)
		}
	}
	for _, want := range []string{"<repo_root>/internal/review/runner.go", "<probe_workdir>/output.txt"} {
		if !strings.Contains(artifact, want) {
			t.Fatalf("probe_results.json missing %q:\n%s", want, artifact)
		}
	}
}

type failingReviewRunArtifactWriter struct {
	err error
}

func newRunnerEvidenceBundleWithWebSearchArtifactForTest(repoRoot string) ReviewEvidenceBundle {
	bundle := newRunnerEvidenceBundleForTest(repoRoot)
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:  true,
		Provider: "gemini",
		Queries:  []ReviewWebSearchEvidenceQuery{{Query: "OpenAI API web_search documentation", Reason: "artifact test"}},
	}
	return bundle
}

func (w failingReviewRunArtifactWriter) WriteReviewRunArtifact(string, []byte) error {
	return w.err
}

func newReviewRunnerWithArtifactWriterForTest(t *testing.T, evidence ReviewEvidenceProvider, probes ReviewProbeExecutor, model ReviewModel, writer ReviewRunArtifactWriter, warningWriter io.Writer) *ReviewRunner {
	t.Helper()

	runner, err := NewReviewRunner(ReviewRunnerOptions{
		EvidenceBuilder:       evidence,
		ProbeRunner:           probes,
		Model:                 model,
		ArtifactWriter:        writer,
		ArtifactWarningWriter: warningWriter,
	})
	if err != nil {
		t.Fatalf("NewReviewRunner() error = %v, want nil", err)
	}
	return runner
}

func assertReviewArtifactsExistForTest(t *testing.T, dir string, names ...string) {
	t.Helper()

	for _, name := range names {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
	}
}

func readReviewArtifactForTest(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return data
}
