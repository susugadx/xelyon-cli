package review

import (
	"reflect"
	"testing"
	"time"
)

func TestBuildReviewProbeSummaries(t *testing.T) {
	results := []ReviewProbeResult{
		{
			ID:     "passed",
			Mode:   ReviewProbeHostReadOnly,
			Status: ReviewProbePassed,
			CommandResults: []ReviewProbeCommandResult{
				{
					Command:  "cat",
					Args:     []string{"keep.txt"},
					WorkDir:  "/repo",
					Status:   ReviewProbePassed,
					ExitCode: 0,
					Duration: 42 * time.Millisecond,
				},
			},
		},
		{
			ID:     "failed",
			Mode:   ReviewProbeScratchOnly,
			Status: ReviewProbeFailed,
			Error:  "probe command failed: go test ./...",
			CommandResults: []ReviewProbeCommandResult{
				{
					Command:         "go",
					Args:            []string{"test", "./..."},
					WorkDir:         "/scratch",
					Status:          ReviewProbeFailed,
					ExitCode:        1,
					OutputTruncated: true,
					Error:           "exit status 1",
					Duration:        2300 * time.Millisecond,
				},
			},
		},
		{
			ID:     "blocked",
			Mode:   ReviewProbeHostReadOnly,
			Status: ReviewProbeBlocked,
			Error:  "probe command blocked: rm -rf .",
		},
		{
			ID:     "timed_out",
			Mode:   ReviewProbeRepoSandbox,
			Status: ReviewProbeTimedOut,
			Error:  "probe command timed out: sleep 10",
		},
		{
			ID:              "mutated",
			Mode:            ReviewProbeRepoSandbox,
			Status:          ReviewProbeMutatedWorktree,
			MutatedWorktree: true,
			MutatedFiles:    []string{"keep.txt", "tmp/new.txt"},
			OutputTruncated: true,
			Error:           "probe command changed the working tree",
		},
	}

	got := BuildReviewProbeSummaries(results)
	if len(got) != len(results) {
		t.Fatalf("len(BuildReviewProbeSummaries()) = %d, want %d", len(got), len(results))
	}

	byID := make(map[string]ReviewProbeSummary, len(got))
	for _, summary := range got {
		byID[summary.ProbeID] = summary
	}

	if byID["passed"].Status != ReviewProbePassed {
		t.Fatalf("passed.Status = %q, want %q", byID["passed"].Status, ReviewProbePassed)
	}
	if byID["failed"].Status != ReviewProbeFailed {
		t.Fatalf("failed.Status = %q, want %q", byID["failed"].Status, ReviewProbeFailed)
	}
	if byID["blocked"].Status != ReviewProbeBlocked {
		t.Fatalf("blocked.Status = %q, want %q", byID["blocked"].Status, ReviewProbeBlocked)
	}
	if byID["timed_out"].Status != ReviewProbeTimedOut {
		t.Fatalf("timed_out.Status = %q, want %q", byID["timed_out"].Status, ReviewProbeTimedOut)
	}
	if byID["mutated"].Status != ReviewProbeMutatedWorktree {
		t.Fatalf("mutated.Status = %q, want %q", byID["mutated"].Status, ReviewProbeMutatedWorktree)
	}

	failed := byID["failed"]
	if len(failed.Commands) != 1 {
		t.Fatalf("len(failed.Commands) = %d, want 1", len(failed.Commands))
	}
	cmd := failed.Commands[0]
	if cmd.Command != "go" {
		t.Fatalf("failed.Commands[0].Command = %q, want go", cmd.Command)
	}
	if cmd.Status != ReviewProbeFailed {
		t.Fatalf("failed.Commands[0].Status = %q, want %q", cmd.Status, ReviewProbeFailed)
	}
	if cmd.ExitCode != 1 {
		t.Fatalf("failed.Commands[0].ExitCode = %d, want 1", cmd.ExitCode)
	}
	if !cmd.OutputTruncated {
		t.Fatal("failed.Commands[0].OutputTruncated = false, want true")
	}
	if cmd.DurationMs != 2300 {
		t.Fatalf("failed.Commands[0].DurationMs = %d, want 2300", cmd.DurationMs)
	}

	mutated := byID["mutated"]
	if !mutated.MutatedWorktree {
		t.Fatal("mutated.MutatedWorktree = false, want true")
	}
	if !reflect.DeepEqual(mutated.MutatedFiles, []string{"keep.txt", "tmp/new.txt"}) {
		t.Fatalf("mutated.MutatedFiles = %#v", mutated.MutatedFiles)
	}
	if !mutated.OutputTruncated {
		t.Fatal("mutated.OutputTruncated = false, want true")
	}
}

func TestNewReviewReportSkeleton(t *testing.T) {
	req := NewCurrentChangesRequest("safety first")
	generatedAt := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

	got := NewReviewReportSkeleton(req, generatedAt)
	if got.SchemaVersion != ReviewReportSchemaVersionV2 {
		t.Fatalf("SchemaVersion = %q, want %q", got.SchemaVersion, ReviewReportSchemaVersionV2)
	}
	if got.TargetKind != TargetCurrentChanges {
		t.Fatalf("TargetKind = %q, want %q", got.TargetKind, TargetCurrentChanges)
	}
	if got.CustomInstructions != "safety first" {
		t.Fatalf("CustomInstructions = %q, want %q", got.CustomInstructions, "safety first")
	}
	if !got.GeneratedAt.Equal(generatedAt) {
		t.Fatalf("GeneratedAt = %s, want %s", got.GeneratedAt, generatedAt)
	}
	if got.OverallVerificationStatus != ReviewVerificationUnverified {
		t.Fatalf("OverallVerificationStatus = %q, want %q", got.OverallVerificationStatus, ReviewVerificationUnverified)
	}
	if got.Verdict != ReviewVerdictBlocked {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, ReviewVerdictBlocked)
	}
	if got.Summary != ReviewReportSkeletonBlockedSummary {
		t.Fatalf("Summary = %q, want %q", got.Summary, ReviewReportSkeletonBlockedSummary)
	}
	if got.RootCauseGroups == nil || len(got.RootCauseGroups) != 0 {
		t.Fatalf("RootCauseGroups = %#v, want non-nil empty slice", got.RootCauseGroups)
	}
	if got.ProbeSummaries == nil || len(got.ProbeSummaries) != 0 {
		t.Fatalf("ProbeSummaries = %#v, want non-nil empty slice", got.ProbeSummaries)
	}
	if got.CheckedSurfaces == nil || len(got.CheckedSurfaces) != 0 {
		t.Fatalf("CheckedSurfaces = %#v, want non-nil empty slice", got.CheckedSurfaces)
	}
	if got.UnverifiedSurfaces == nil || len(got.UnverifiedSurfaces) != 0 {
		t.Fatalf("UnverifiedSurfaces = %#v, want non-nil empty slice", got.UnverifiedSurfaces)
	}
	if got.ResidualRisks == nil || len(got.ResidualRisks) != 0 {
		t.Fatalf("ResidualRisks = %#v, want non-nil empty slice", got.ResidualRisks)
	}
	if err := ValidateReviewReport(got); err != nil {
		t.Fatalf("ValidateReviewReport(skeleton) error = %v, want nil", err)
	}
}
