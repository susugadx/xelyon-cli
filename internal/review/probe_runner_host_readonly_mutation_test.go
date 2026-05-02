package review

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestProbeRunner_HostReadOnlyMutationDetected(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:             "probe-mutation",
		Mode:           ReviewProbeHostReadOnly,
		Timeout:        10 * time.Second,
		MaxOutputBytes: 1024,
		Commands: []ReviewProbeCommand{
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"}},
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
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutate$"}},
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
			{Command: "go", Args: []string{"test", "-count=1", "./probe", "-run", "^TestProbeMutateDirtyExistingPath$"}},
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
