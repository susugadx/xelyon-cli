package review

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewRunnerRunUsesTrustedProbeSummaries(t *testing.T) {
	repoRoot := t.TempDir()
	probeRoot := filepath.Join(t.TempDir(), reviewprobe.ReviewProbeSandboxTempPrefix+"trusted")
	probeWorkDir := filepath.Join(probeRoot, "worktree")
	repoFile := filepath.Join(repoRoot, "internal/review/runner.go")
	probeFile := filepath.Join(probeWorkDir, "output.txt")
	evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest(repoRoot)}
	probeResult := reviewprobe.ReviewProbeResult{
		ID:              "probe-1",
		Mode:            domain.ReviewProbeHostReadOnly,
		Status:          domain.ReviewProbeFailed,
		MutatedFiles:    []string{repoFile, probeFile},
		OutputTruncated: true,
		Error:           "probe failed at " + repoFile + " and " + probeFile,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command:         "cat " + probeFile,
				Args:            []string{repoFile, probeFile},
				WorkDir:         probeWorkDir,
				Status:          domain.ReviewProbeFailed,
				ExitCode:        1,
				OutputTruncated: true,
				Error:           "exit status 1 at " + probeFile,
				Duration:        1500 * time.Millisecond,
			},
		},
	}
	modelReport := newRunnerBlockedReportForTest([]reviewreport.ReviewProbeSummary{
		{
			ProbeID:      "fake-probe",
			Mode:         domain.ReviewProbeHostReadOnly,
			Status:       domain.ReviewProbePassed,
			MutatedFiles: []string{"/fake/model/path"},
			Error:        "fake model summary must be ignored",
		},
	})
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	wantSummaries := []reviewreport.ReviewProbeSummary{
		{
			ProbeID:         "probe-1",
			Mode:            domain.ReviewProbeHostReadOnly,
			Status:          domain.ReviewProbeFailed,
			MutatedFiles:    []string{"internal/review/runner.go", "<probe_workdir>/output.txt"},
			OutputTruncated: true,
			Error:           "probe failed at <repo_root>/internal/review/runner.go and <probe_workdir>/output.txt",
			Commands: []reviewreport.ReviewProbeCommandSummary{
				{
					Command:         "cat <probe_workdir>/output.txt",
					Args:            []string{"<repo_root>/internal/review/runner.go", "<probe_workdir>/output.txt"},
					WorkDir:         "<probe_workdir>",
					Status:          domain.ReviewProbeFailed,
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
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command: "cat " + repoFile,
				Args:    []string{repoFile},
				WorkDir: repoRoot,
				Status:  domain.ReviewProbePassed,
			},
		},
	}
	modelReport := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")
	modelReport.CheckedSurfaces = []reviewreport.ReviewSurfaceCoverage{
		{
			SurfaceID: "surface-1",
			EvidenceRefs: []reviewreport.ReviewEvidenceRef{
				{
					Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
					ProbeID:      "probe-1",
					CommandIndex: reviewreport.ReviewCommandIndex(0),
				},
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
	model := &runnerFakeModel{
		responses: []runnerFakeModelResponse{
			{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
			{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
			saturatedRunnerModelResponseForTest(t),
		},
	}
	runner := newReviewRunnerForTest(t, evidence, probes, model)

	got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
	if err != nil {
		t.Fatalf("Run() error = %v, want nil", err)
	}
	wantSummaries := []reviewreport.ReviewProbeSummary{
		{
			ProbeID:      "probe-1",
			Mode:         domain.ReviewProbeHostReadOnly,
			Status:       domain.ReviewProbePassed,
			MutatedFiles: []string{},
			Commands: []reviewreport.ReviewProbeCommandSummary{
				{
					Command: "cat <repo_root>/internal/review/runner.go",
					Args:    []string{"<repo_root>/internal/review/runner.go"},
					WorkDir: ".",
					Status:  domain.ReviewProbePassed,
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
	probeResult := reviewprobe.ReviewProbeResult{
		ID:     "probe-1",
		Mode:   domain.ReviewProbeHostReadOnly,
		Status: domain.ReviewProbePassed,
		CommandResults: []reviewprobe.ReviewProbeCommandResult{
			{
				Command: "go",
				Status:  domain.ReviewProbePassed,
			},
		},
	}
	modelReport := newRunnerCleanReportForTest([]reviewreport.ReviewProbeSummary{
		{
			ProbeID: "fake-probe",
			Mode:    domain.ReviewProbeHostReadOnly,
			Status:  domain.ReviewProbePassed,
			Commands: []reviewreport.ReviewProbeCommandSummary{
				{
					Command: "go",
					Status:  domain.ReviewProbePassed,
				},
			},
		},
	})
	modelReport.CheckedSurfaces = []reviewreport.ReviewSurfaceCoverage{
		{
			SurfaceID: "surface-1",
			EvidenceRefs: []reviewreport.ReviewEvidenceRef{
				{
					Kind:         reviewreport.ReviewEvidenceKindProbeCommand,
					ProbeID:      "fake-probe",
					CommandIndex: reviewreport.ReviewCommandIndex(0),
				},
			},
		},
	}
	probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
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
