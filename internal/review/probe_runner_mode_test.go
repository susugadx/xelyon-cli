package review

import (
	"context"
	"strings"
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

func TestProbeRunner_RepoSandboxIsExecuted(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-repo-sandbox-mode",
		Mode:           ReviewProbeRepoSandbox,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"keep.txt"}},
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
	if !strings.Contains(result.CommandResults[0].Output, "keep") {
		t.Fatalf("CommandResults[0].Output = %q, want copied repo file output", result.CommandResults[0].Output)
	}
}
