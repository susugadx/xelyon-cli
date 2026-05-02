package review

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestProbeRunner_ScratchOnlyIsExecuted(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-scratch-mode",
		Mode:           ReviewProbeScratchOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Files: []ReviewProbeFile{
			{Path: "hello.txt", Content: "ok\n"},
		},
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"hello.txt"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
}

func TestProbeRunner_RepoSandboxUnsupportedMode(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:   "probe-unsupported",
		Mode: ReviewProbeRepoSandbox,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want unsupported mode error")
	}
	if !errors.Is(err, ErrUnsupportedReviewProbeMode) {
		t.Fatalf("Run() error = %v, want ErrUnsupportedReviewProbeMode", err)
	}
	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
}
