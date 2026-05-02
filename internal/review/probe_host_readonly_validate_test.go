package review

import (
	"context"
	"strings"
	"testing"
)

func TestHostReadOnlyValidateRequest_AppliesDefaultExecutionLimits(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	executor := newHostReadOnlyExecutor(repo)

	normalized, err := executor.validateRequest(ReviewProbeRequest{
		ID:   "probe-default-limits-validate",
		Mode: ReviewProbeHostReadOnly,
		Commands: []ReviewProbeCommand{
			{
				Command: "git",
				Args:    []string{"status", "--short"},
			},
		},
	})
	if err != nil {
		t.Fatalf("validateRequest() error = %v", err)
	}
	if normalized.timeout != defaultReviewProbeTimeout {
		t.Fatalf("timeout = %v, want %v", normalized.timeout, defaultReviewProbeTimeout)
	}
	if normalized.maxOutputBytes != defaultReviewProbeMaxOutputBytes {
		t.Fatalf("maxOutputBytes = %d, want %d", normalized.maxOutputBytes, defaultReviewProbeMaxOutputBytes)
	}
}

func TestHostReadOnlyRun_DirectEntryAppliesDefaultExecutionLimits(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	executor := newHostReadOnlyExecutor(repo)

	writeTestFile(t, repo+"/huge.txt", strings.Repeat("x", int(defaultReviewProbeMaxOutputBytes)+2048))

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "probe-default-limits-run",
		Mode: ReviewProbeHostReadOnly,
		Commands: []ReviewProbeCommand{
			{
				Command: "cat",
				Args:    []string{"huge.txt"},
			},
		},
	})

	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if !result.OutputTruncated {
		t.Fatal("OutputTruncated = false, want true")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !result.CommandResults[0].OutputTruncated {
		t.Fatal("CommandResults[0].OutputTruncated = false, want true")
	}
	if int64(len(result.CommandResults[0].Output)) != defaultReviewProbeMaxOutputBytes {
		t.Fatalf("len(CommandResults[0].Output) = %d, want %d", len(result.CommandResults[0].Output), defaultReviewProbeMaxOutputBytes)
	}
}
