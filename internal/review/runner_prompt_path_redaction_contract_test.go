package review

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewRunnerPromptRedactsAbsoluteRepoRoot(t *testing.T) {
	repoRoot := t.TempDir()
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest(repoRoot)}
	probes := &runnerFakeProbeRunner{}
	plan := newRunnerNoProbePlanForTest()
	report := newRunnerCleanReportForTest([]reviewreport.ReviewProbeSummary{})
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
	if got, want := len(model.requests), 3; got != want {
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
	probeRoot := filepath.Join(t.TempDir(), reviewprobe.ReviewProbeSandboxTempPrefix+"abc")
	probeWorkDir := filepath.Join(probeRoot, "worktree")
	scratchRoot := filepath.Join(t.TempDir(), reviewprobe.ReviewProbeScratchTempPrefix+"def")
	repoFile := filepath.Join(repoRoot, "internal/review/runner.go")
	probeWorkFile := filepath.Join(probeWorkDir, "output.txt")
	probeRuntimeFile := filepath.Join(probeRoot, "runtime/home/output.txt")
	scratchFile := filepath.Join(scratchRoot, "tmp/mutated.txt")
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest(repoRoot)}
	probeResult := reviewprobe.ReviewProbeResult{
		ID:              "probe-1",
		Mode:            domain.ReviewProbeHostReadOnly,
		Status:          domain.ReviewProbeFailed,
		MutatedWorktree: true,
		MutatedFiles:    []string{repoFile, filepath.Join(probeWorkDir, "mutated.txt"), scratchFile},
		Error:           "probe failed at " + repoFile + " using " + probeRuntimeFile,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command: "pwd",
				WorkDir: repoRoot,
				Status:  domain.ReviewProbeFailed,
				Output:  "repo path: " + repoFile,
				Error:   "repo error: " + repoRoot,
			},
			{
				Command: "cat",
				Args:    []string{probeWorkFile, scratchFile},
				WorkDir: probeWorkDir,
				Status:  domain.ReviewProbeFailed,
				Output:  "probe path: " + probeWorkFile,
				Error:   "probe runtime error: " + probeRuntimeFile,
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerBlockedReportForTest(nil)))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	if got, want := len(model.requests), 3; got != want {
		t.Fatalf("model requests = %d, want %d", got, want)
	}

	for i, prompt := range []string{model.requests[1].Prompt, model.requests[2].Prompt} {
		for _, leaked := range []string{repoRoot, repoFile, probeRoot, probeWorkDir, probeWorkFile, probeRuntimeFile, scratchRoot, scratchFile} {
			if strings.Contains(prompt, leaked) {
				t.Fatalf("prompt %d contains absolute path %q:\n%s", i+1, leaked, prompt)
			}
		}
	}
	secondPrompt := model.requests[1].Prompt
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
	if got, want := got.ProbeSummaries[0].Status, domain.ReviewProbeMutatedWorktree; got != want {
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
