package review

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeRunner_HostReadOnlyPassed(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-pass",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"status", "--short"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != ReviewProbePassed {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, ReviewProbePassed)
	}
}

func TestProbeRunner_HostReadOnlyPassed_GitGlobalOption(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-pass-git-global-option",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "git", Args: []string{"--no-optional-locks", "status", "--short"}},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if result.MutatedWorktree {
		t.Fatalf("MutatedWorktree = true, want false")
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != ReviewProbePassed {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, ReviewProbePassed)
	}
}

func TestProbeRunner_HostReadOnlyTimedOut(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-timeout",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        100 * time.Millisecond,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "TestProbeSleep"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeTimedOut {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeTimedOut, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if result.CommandResults[0].Status != ReviewProbeTimedOut {
		t.Fatalf("CommandResults[0].Status = %q, want %q", result.CommandResults[0].Status, ReviewProbeTimedOut)
	}
}

func TestProbeRunner_HostReadOnlyOutputTruncated(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-truncate",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 32,
		Commands: []ReviewProbeCommand{
			{
				Command: "cat",
				Args:    []string{"large.txt"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbePassed)
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
	if len(result.CommandResults[0].Output) > 32 {
		t.Fatalf("len(CommandResults[0].Output) = %d, want <= 32", len(result.CommandResults[0].Output))
	}
}

func TestProbeRunner_HostReadOnlyMutationDetected(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error)
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("probe/probe_generated.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain probe/probe_generated.txt", result.MutatedFiles)
	}
}

func TestProbeRunner_HostReadOnlyMutationDetected_DirtyWorktreeReportsOnlyDelta(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	writeTestFile(t, filepath.Join(repo, "keep.txt"), "pre-existing-dirty\n")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation-dirty",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error)
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("probe/probe_generated.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain probe/probe_generated.txt", result.MutatedFiles)
	}
	if containsString(result.MutatedFiles, filepath.ToSlash("keep.txt")) {
		t.Fatalf("MutatedFiles = %#v, keep.txt should not be included", result.MutatedFiles)
	}
}

func TestProbeRunner_HostReadOnlyMutationDetected_DirtyExistingPathChanged(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	writeTestFile(t, filepath.Join(repo, "keep.txt"), "pre-existing-dirty\n")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation-dirty-existing-path",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "go",
				Args:    []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutateDirtyExistingPath$"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeMutatedWorktree {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeMutatedWorktree, result.Error)
	}
	if !result.MutatedWorktree {
		t.Fatal("MutatedWorktree = false, want true")
	}
	if !containsString(result.MutatedFiles, filepath.ToSlash("keep.txt")) {
		t.Fatalf("MutatedFiles = %#v, want to contain keep.txt", result.MutatedFiles)
	}
}

func TestProbeRunner_HostReadOnlyArgsDoNotUseShell(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)
	injectedPath := filepath.Join(repo, "shell_pwned")

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-no-shell",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        2 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{
				Command: "git",
				Args:    []string{"status", "--short; touch shell_pwned"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != ReviewProbeFailed {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeFailed)
	}
	if _, err := os.Stat(injectedPath); !os.IsNotExist(err) {
		t.Fatalf("shell-like argument should not create %s, stat error = %v", injectedPath, err)
	}
}
