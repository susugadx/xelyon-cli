package review

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewpromptreduction "github.com/susugadx/xelyon-cli/internal/review/promptreduction"
)

func TestReviewRunnerPromptReductionModeControlsProbeOutputCompaction(t *testing.T) {
	tests := []struct {
		name              string
		mode              reviewpromptreduction.ReviewPromptReductionMode
		wantPromptCompact bool
		wantReplaced      int
	}{
		{
			name:              "apply compacts provider prompt",
			mode:              reviewpromptreduction.ReviewPromptReductionModeApply,
			wantPromptCompact: true,
			wantReplaced:      1,
		},
		{
			name:              "dry run records candidate without compacting prompt",
			mode:              reviewpromptreduction.ReviewPromptReductionModeDryRun,
			wantPromptCompact: false,
			wantReplaced:      0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
			probeResult := reviewprobe.ReviewProbeResult{
				ID:     "probe-1",
				Mode:   domain.ReviewProbeHostReadOnly,
				Status: domain.ReviewProbePassed,
				CommandResults: []reviewprobe.ReviewProbeCommandResult{
					{
						Command: "go",
						Args:    []string{"test", "./..."},
						Status:  domain.ReviewProbePassed,
						Output: strings.Join([]string{
							"ok   github.com/susugadx/xelyon-cli/internal/review 0.123s",
							strings.Repeat("verbose detail that should not be sent after compaction\n", 80),
						}, "\n"),
					},
				},
			}
			probes := &runnerFakeProbeRunner{results: map[string]reviewprobe.ReviewProbeResult{"probe-1": probeResult}}
			modelReport := newRunnerCleanReportWithPassedProbeEvidenceForTest("probe-1")
			modelReport.ProbeSummaries = newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, []reviewprobe.ReviewProbeResult{probeResult})
			model := &runnerFakeModel{
				responses: []runnerFakeModelResponse{
					{content: string(mustMarshalReviewProbePlanForRunnerTest(t, newRunnerProbePlanForTest("probe-1")))},
					{content: string(mustMarshalReviewReportForRunnerTest(t, modelReport))},
					saturatedRunnerModelResponseForTest(t),
				},
			}
			runner, err := NewReviewRunner(ReviewRunnerOptions{
				EvidenceBuilder:     evidence,
				ProbeRunner:         probes,
				Model:               model,
				PromptReductionMode: tt.mode,
			})
			if err != nil {
				t.Fatalf("NewReviewRunner() error = %v, want nil", err)
			}

			if _, err := runner.Run(context.Background(), NewCurrentChangesRequest("")); err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}

			reportPrompt := model.requests[1].Prompt
			if tt.wantPromptCompact {
				if !strings.Contains(reportPrompt, "## Review State Summary") ||
					!strings.Contains(reportPrompt, "target: current_changes") ||
					!strings.Contains(reportPrompt, "impact_surfaces:") {
					t.Fatalf("report prompt missing review state summary:\n%s", reportPrompt)
				}
				if !strings.Contains(reportPrompt, "omitted old successful validation command output") ||
					!strings.Contains(reportPrompt, `"output_compacted": true`) {
					t.Fatalf("report prompt missing compact placeholder:\n%s", reportPrompt)
				}
				if strings.Contains(reportPrompt, "verbose detail that should not be sent") {
					t.Fatalf("report prompt leaked raw verbose detail:\n%s", reportPrompt)
				}
			} else {
				if strings.Contains(reportPrompt, "## Review State Summary") {
					t.Fatalf("dry-run report prompt changed with review state summary:\n%s", reportPrompt)
				}
				if !strings.Contains(reportPrompt, "verbose detail that should not be sent after compaction") {
					t.Fatalf("dry-run report prompt missing raw verbose detail:\n%s", reportPrompt)
				}
			}

			reductionReport := runner.PromptReductionReport()
			if reductionReport.CandidateCount != 2 || reductionReport.ReplacedCount != tt.wantReplaced ||
				reductionReport.ClassifierCounts["validation"] != 1 ||
				reductionReport.ClassifierCounts["probe_result_absorption_candidate"] != 1 ||
				reductionReport.KeptReasonCounts[reviewpromptreduction.ReviewProbeRawOutputReasonArtifactMissing] != 1 {
				t.Fatalf("PromptReductionReport() = %#v, want validation replacement and blocked probe candidate with %d replacements", reductionReport, tt.wantReplaced)
			}
			if tt.wantPromptCompact {
				if reductionReport.StateSummaryCount != 2 || reductionReport.AbsorbedItemCount != 2 || !reductionReport.QualityFloorPreserved {
					t.Fatalf("PromptReductionReport() = %#v, want state summaries with preserved quality floor", reductionReport)
				}
			}
		})
	}
}
