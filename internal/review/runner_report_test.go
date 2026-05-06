package review

import "testing"

func TestFinalizeReviewRunnerReportDowngradesCleanReportWithBlockedTrustedProbe(t *testing.T) {
	tests := []struct {
		name            string
		status          ReviewProbeStatus
		mutatedWorktree bool
		wantMutation    bool
	}{
		{name: "blocked", status: ReviewProbeBlocked},
		{name: "timed out", status: ReviewProbeTimedOut},
		{name: "mutated worktree", status: ReviewProbeMutatedWorktree, wantMutation: true},
		{name: "mutated worktree flag", status: ReviewProbeFailed, mutatedWorktree: true, wantMutation: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := finalizeReviewRunnerReport(newRunnerCleanReportForTest(nil), []ReviewProbeSummary{
				{
					ProbeID:         "probe-1",
					Mode:            ReviewProbeHostReadOnly,
					Status:          tt.status,
					MutatedWorktree: tt.mutatedWorktree,
				},
			})
			if err != nil {
				t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
			}
			if got.Verdict != ReviewVerdictBlocked {
				t.Fatalf("Verdict = %q, want %q", got.Verdict, ReviewVerdictBlocked)
			}
			if got.OverallVerificationStatus != ReviewVerificationBlockedOrInconclusive {
				t.Fatalf("OverallVerificationStatus = %q, want %q", got.OverallVerificationStatus, ReviewVerificationBlockedOrInconclusive)
			}
			if tt.wantMutation && got.ProbeSummaries[0].Status != ReviewProbeMutatedWorktree {
				t.Fatalf("ProbeSummaries[0].Status = %q, want %q", got.ProbeSummaries[0].Status, ReviewProbeMutatedWorktree)
			}
			if tt.wantMutation && !got.ProbeSummaries[0].MutatedWorktree {
				t.Fatal("ProbeSummaries[0].MutatedWorktree = false, want true")
			}
		})
	}
}

func TestFinalizeReviewRunnerReportDowngradesVerifiedFindingsWithBlockedTrustedProbe(t *testing.T) {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)

	got, err := finalizeReviewRunnerReport(report, []ReviewProbeSummary{
		{
			ProbeID: "probe-1",
			Mode:    ReviewProbeHostReadOnly,
			Status:  ReviewProbeBlocked,
		},
	})
	if err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
	if got.Verdict != ReviewVerdictHasFindings {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, ReviewVerdictHasFindings)
	}
	if got.OverallVerificationStatus != ReviewVerificationPartiallyVerified {
		t.Fatalf("OverallVerificationStatus = %q, want %q", got.OverallVerificationStatus, ReviewVerificationPartiallyVerified)
	}
}
