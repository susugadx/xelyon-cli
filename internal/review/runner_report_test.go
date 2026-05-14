package review

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

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
			got, err := finalizeReviewRunnerReport(newRunnerCleanReportForTest(nil), newRunnerProbePlanForTest("probe-1"), []ReviewProbeSummary{
				{
					ProbeID:         "probe-1",
					Mode:            ReviewProbeHostReadOnly,
					Status:          tt.status,
					MutatedWorktree: tt.mutatedWorktree,
				},
			}, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
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
	report.ScopeCoverage = &ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: ReviewReportImpactSurfaceChecked},
		},
		ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
			{RiskID: "risk-1", Status: ReviewReportCandidateRiskFinding, FindingIDs: []string{"finding-1"}},
		},
	}

	got, err := finalizeReviewRunnerReport(report, newRunnerProbePlanForTest("probe-1"), []ReviewProbeSummary{
		{
			ProbeID: "probe-1",
			Mode:    ReviewProbeHostReadOnly,
			Status:  ReviewProbeBlocked,
		},
	}, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
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

func TestFinalizeReviewRunnerReportInjectsRedactedTrustedProbeSummaries(t *testing.T) {
	repoRoot := t.TempDir()
	probeRoot := filepath.Join(t.TempDir(), reviewProbeSandboxTempPrefix+"finalizer")
	probeWorkDir := filepath.Join(probeRoot, "worktree")
	repoFile := filepath.Join(repoRoot, "internal/review/runner.go")
	probeFile := filepath.Join(probeWorkDir, "raw-output.txt")
	trustedSummaries := []ReviewProbeSummary{
		{
			ProbeID:         "probe-raw",
			Mode:            ReviewProbeHostReadOnly,
			Status:          ReviewProbeFailed,
			MutatedFiles:    []string{repoFile, probeFile},
			OutputTruncated: true,
			Error:           "raw paths " + repoFile + " " + probeFile,
			Commands: []ReviewProbeCommandSummary{
				{
					Command:         "cat " + probeFile,
					Args:            []string{repoFile, probeFile},
					WorkDir:         probeWorkDir,
					Status:          ReviewProbeFailed,
					ExitCode:        1,
					OutputTruncated: true,
					Error:           "failed at " + probeFile,
					DurationMs:      25,
				},
			},
		},
	}
	original := cloneReviewProbeSummariesForRedactionTest(trustedSummaries)
	redactor := newRunnerReportRedactorForTest(t, repoRoot, []ReviewProbeResult{
		{
			ID:           "probe-raw",
			Mode:         ReviewProbeHostReadOnly,
			Status:       ReviewProbeFailed,
			MutatedFiles: []string{repoFile, probeFile},
			Error:        "raw paths " + repoFile + " " + probeFile,
			CommandResults: []ReviewProbeCommandResult{
				{
					Command: "cat " + probeFile,
					Args:    []string{repoFile, probeFile},
					WorkDir: probeWorkDir,
					Status:  ReviewProbeFailed,
					Error:   "failed at " + probeFile,
				},
			},
		},
	})

	got, err := finalizeReviewRunnerReport(newRunnerCleanReportForTest(nil), newRunnerProbePlanForTest("probe-1"), trustedSummaries, redactor)
	if err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
	if !reflect.DeepEqual(trustedSummaries, original) {
		t.Fatalf("finalizeReviewRunnerReport() mutated trusted summaries:\ngot  %#v\nwant %#v", trustedSummaries, original)
	}
	if strings.Contains(got.ProbeSummaries[0].Error, repoRoot) || strings.Contains(got.ProbeSummaries[0].Error, probeRoot) {
		t.Fatalf("ProbeSummaries[0].Error leaked raw path: %q", got.ProbeSummaries[0].Error)
	}
	wantMutatedFiles := []string{"internal/review/runner.go", "<probe_workdir>/raw-output.txt"}
	if !reflect.DeepEqual(got.ProbeSummaries[0].MutatedFiles, wantMutatedFiles) {
		t.Fatalf("MutatedFiles = %#v, want %#v", got.ProbeSummaries[0].MutatedFiles, wantMutatedFiles)
	}
	if got.ProbeSummaries[0].Commands[0].Command != "cat <probe_workdir>/raw-output.txt" {
		t.Fatalf("command Command = %q, want redacted command", got.ProbeSummaries[0].Commands[0].Command)
	}
	wantArgs := []string{"<repo_root>/internal/review/runner.go", "<probe_workdir>/raw-output.txt"}
	if !reflect.DeepEqual(got.ProbeSummaries[0].Commands[0].Args, wantArgs) {
		t.Fatalf("command Args = %#v, want %#v", got.ProbeSummaries[0].Commands[0].Args, wantArgs)
	}
	if got.ProbeSummaries[0].Commands[0].WorkDir != "<probe_workdir>" {
		t.Fatalf("command WorkDir = %q, want redacted workdir", got.ProbeSummaries[0].Commands[0].WorkDir)
	}
	if got.ProbeSummaries[0].Commands[0].Error != "failed at <probe_workdir>/raw-output.txt" {
		t.Fatalf("command Error = %q, want redacted error", got.ProbeSummaries[0].Commands[0].Error)
	}
}

func TestFinalizeReviewRunnerReportKeepsEmptyTrustedProbeSummariesNil(t *testing.T) {
	got, err := finalizeReviewRunnerReport(newRunnerCleanReportForTest(nil), newRunnerProbePlanForTest("probe-1"), nil, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
	if err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
	if got.ProbeSummaries != nil {
		t.Fatalf("ProbeSummaries = %#v, want nil", got.ProbeSummaries)
	}
}

func TestFinalizeReviewRunnerReportModelOutputRejectsComputedSummary(t *testing.T) {
	report := newRunnerCleanReportForTest(nil)
	report.ComputedSummary = &ReviewReportComputedSummary{FindingCount: 99}
	data := mustMarshalReviewReportForRunnerTest(t, report)

	_, err := finalizeReviewRunnerReportModelOutput(string(data), newRunnerProbePlanForTest("probe-1"), nil, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
	if err == nil {
		t.Fatal("finalizeReviewRunnerReportModelOutput() error = nil, want computed_summary rejection")
	}
	if !strings.Contains(err.Error(), "computed_summary") {
		t.Fatalf("finalizeReviewRunnerReportModelOutput() error = %q, want computed_summary", err.Error())
	}
}

func TestFinalizeReviewRunnerReportComputesSummaryForCleanReport(t *testing.T) {
	got, err := finalizeReviewRunnerReport(newRunnerCleanReportForTest(nil), newRunnerProbePlanForTest("probe-1"), nil, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
	if err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
	assertReviewReportComputedSummaryPointerForTest(t, got.ComputedSummary, ReviewReportComputedSummary{
		CheckedSurfaceCount: 1,
		CandidateRiskCount:  1,
		DismissedRiskCount:  1,
	})
}

func TestFinalizeReviewRunnerReportComputesSummaryForFindingRisk(t *testing.T) {
	got, err := finalizeReviewRunnerReport(newPlanAwareHasFindingsReportForValidationTest(), newRunnerProbePlanForTest("probe-1"), nil, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
	if err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
	assertReviewReportComputedSummaryPointerForTest(t, got.ComputedSummary, ReviewReportComputedSummary{
		RootCauseGroupCount: 1,
		FindingCount:        1,
		CheckedSurfaceCount: 1,
		CandidateRiskCount:  1,
		FindingRiskCount:    1,
	})
}

func TestFinalizeReviewRunnerReportComputesBlockedProbeCount(t *testing.T) {
	got, err := finalizeReviewRunnerReport(newRunnerCleanReportForTest(nil), newRunnerProbePlanForTest("probe-1"), []ReviewProbeSummary{
		{
			ProbeID: "probe-1",
			Mode:    ReviewProbeHostReadOnly,
			Status:  ReviewProbeBlocked,
		},
	}, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
	if err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
	assertReviewReportComputedSummaryPointerForTest(t, got.ComputedSummary, ReviewReportComputedSummary{
		CheckedSurfaceCount: 1,
		CandidateRiskCount:  1,
		DismissedRiskCount:  1,
		ProbeCount:          1,
		BlockedProbeCount:   1,
	})
}

func TestFinalizeReviewRunnerReportOverwritesPreexistingComputedSummary(t *testing.T) {
	report := newRunnerCleanReportForTest(nil)
	report.ComputedSummary = &ReviewReportComputedSummary{
		FindingCount:              99,
		MutatedWorktreeProbeCount: 99,
	}

	got, err := finalizeReviewRunnerReport(report, newRunnerProbePlanForTest("probe-1"), nil, newRunnerReportRedactorForTest(t, "/tmp/review-runner/repo", nil))
	if err != nil {
		t.Fatalf("finalizeReviewRunnerReport() error = %v, want nil", err)
	}
	assertReviewReportComputedSummaryPointerForTest(t, got.ComputedSummary, ReviewReportComputedSummary{
		CheckedSurfaceCount: 1,
		CandidateRiskCount:  1,
		DismissedRiskCount:  1,
	})
}

func newRunnerReportRedactorForTest(t *testing.T, repoRoot string, probeResults []ReviewProbeResult) reviewRunnerPromptRedactor {
	t.Helper()

	return newReviewRunnerPromptRedactor(newRunnerEvidenceBundleForTest(repoRoot), probeResults)
}
