package review

import (
	"context"
	"reflect"
	"strings"
	"testing"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func TestReviewRunnerRunStopsProbeExecutionAfterMutatedWorktree(t *testing.T) {
	tests := []struct {
		name          string
		mutatedResult ReviewProbeResult
	}{
		{
			name: "status mutated worktree",
			mutatedResult: ReviewProbeResult{
				ID:           "probe-a",
				Mode:         ReviewProbeHostReadOnly,
				Status:       ReviewProbeMutatedWorktree,
				MutatedFiles: []string{"internal/review/runner.go"},
				Error:        "probe command changed the working tree",
			},
		},
		{
			name: "mutated worktree flag with failed status",
			mutatedResult: ReviewProbeResult{
				ID:              "probe-a",
				Mode:            ReviewProbeHostReadOnly,
				Status:          ReviewProbeFailed,
				MutatedWorktree: true,
				MutatedFiles:    []string{"internal/review/runner.go"},
				Error:           "probe command changed the working tree",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := &runnerFakeEvidenceBuilder{bundle: newRunnerEvidenceBundleForTest("/tmp/review-runner/repo")}
			probes := &runnerFakeProbeRunner{
				results: map[string]ReviewProbeResult{"probe-a": tt.mutatedResult},
			}
			plan := newRunnerProbePlanForTest("probe-a", "probe-b", "probe-c")
			expectedMutatedResult := tt.mutatedResult
			expectedMutatedResult.Status = ReviewProbeMutatedWorktree
			expectedMutatedResult.MutatedWorktree = true
			expectedProbeResults := []ReviewProbeResult{
				expectedMutatedResult,
				{
					ID:     "probe-b",
					Mode:   ReviewProbeHostReadOnly,
					Status: ReviewProbeBlocked,
					Error:  reviewprobe.ProbeSkippedAfterMutationError("probe-a"),
				},
				{
					ID:     "probe-c",
					Mode:   ReviewProbeHostReadOnly,
					Status: ReviewProbeBlocked,
					Error:  reviewprobe.ProbeSkippedAfterMutationError("probe-a"),
				},
			}
			expectedSummaries := newRedactedRunnerProbeSummariesForTest(t, evidence.bundle, expectedProbeResults)
			model := &runnerFakeModel{
				responses: []runnerFakeModelResponse{
					{content: string(mustMarshalReviewProbePlanForRunnerTest(t, plan))},
					{content: string(mustMarshalReviewReportForRunnerTest(t, newRunnerBlockedReportForTest(expectedSummaries)))},
					saturatedRunnerModelResponseForTest(t),
				},
			}
			runner := newReviewRunnerForTest(t, evidence, probes, model)

			got, err := runner.Run(context.Background(), NewCurrentChangesRequest(""))
			if err != nil {
				t.Fatalf("Run() error = %v, want nil", err)
			}
			if got, want := len(probes.calls), 1; got != want {
				t.Fatalf("probe calls = %d, want %d", got, want)
			}
			if got, want := probes.calls[0].ID, "probe-a"; got != want {
				t.Fatalf("first probe call = %q, want %q", got, want)
			}
			if !reflect.DeepEqual(got.ProbeSummaries, expectedSummaries) {
				t.Fatalf("Run() probe summaries = %#v, want %#v", got.ProbeSummaries, expectedSummaries)
			}
			if got, want := got.ProbeSummaries[1].Status, ReviewProbeBlocked; got != want {
				t.Fatalf("skipped probe status = %q, want %q", got, want)
			}
			if !strings.Contains(got.ProbeSummaries[1].Error, "probe-a") {
				t.Fatalf("skipped probe error = %q, want mutated probe ID", got.ProbeSummaries[1].Error)
			}
			if got, want := len(model.requests), 3; got != want {
				t.Fatalf("model requests = %d, want %d", got, want)
			}
			secondPrompt := model.requests[1].Prompt
			for _, want := range []string{
				`"probe_id": "probe-a"`,
				`"status": "mutated_worktree"`,
				`"mutated_worktree": true`,
				`"probe_id": "probe-b"`,
				`"status": "blocked"`,
				`probe skipped because probe \"probe-a\" mutated the working tree`,
			} {
				if !strings.Contains(secondPrompt, want) {
					t.Fatalf("Pass2 prompt missing %q:\n%s", want, secondPrompt)
				}
			}
		})
	}
}
