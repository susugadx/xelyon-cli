package review

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestScratchOnlyExecutor_ResolvesCommandFromSafePath(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	safeBin := filepath.Join(t.TempDir(), "safe-bin")
	createProbeTestScriptCommand(t, safeBin, "cat", "echo safe-cat")

	executor := newScratchOnlyExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + safeBin,
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "scratch-resolve-safe",
		Mode: ReviewProbeScratchOnly,
		Files: []ReviewProbeFile{
			{Path: "check.txt", Content: "ok\n"},
		},
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"check.txt"}},
		},
	})

	if result.Status != ReviewProbePassed {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbePassed, result.Error)
	}
	if len(result.CommandResults) != 1 {
		t.Fatalf("len(CommandResults) = %d, want 1", len(result.CommandResults))
	}
	if !strings.Contains(result.CommandResults[0].Output, "safe-cat") {
		t.Fatalf("CommandResults[0].Output = %q, want to contain safe-cat", result.CommandResults[0].Output)
	}
}

func TestScratchOnlyExecutor_BlocksCommandResolvedInsideScratchDir(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	scratchDir := filepath.Join(t.TempDir(), "scratch-root")
	scratchBin := filepath.Join(scratchDir, "bin")
	safeBin := filepath.Join(t.TempDir(), "safe-bin")

	createProbeTestScriptCommand(t, scratchBin, "cat", "echo scratch-cat")
	createProbeTestScriptCommand(t, safeBin, "cat", "echo safe-cat")

	executor := newScratchOnlyExecutor(repo)
	executor.mktemp = func(dir, pattern string) (string, error) {
		return scratchDir, nil
	}
	executor.baseEnv = []string{
		"PATH=" + strings.Join([]string{scratchBin, safeBin}, string(filepath.ListSeparator)),
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "scratch-resolve-blocked-scratch-bin",
		Mode: ReviewProbeScratchOnly,
		Files: []ReviewProbeFile{
			{Path: "check.txt", Content: "ok\n"},
		},
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"check.txt"}},
		},
	})

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
}

func TestScratchOnlyExecutor_BlocksCommandResolvedInsideRepoRoot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	repoBin := filepath.Join(repo, "bin")
	safeBin := filepath.Join(t.TempDir(), "safe-bin")

	createProbeTestScriptCommand(t, repoBin, "cat", "echo repo-cat")
	createProbeTestScriptCommand(t, safeBin, "cat", "echo safe-cat")

	executor := newScratchOnlyExecutor(repo)
	executor.baseEnv = []string{
		"PATH=" + strings.Join([]string{repoBin, safeBin}, string(filepath.ListSeparator)),
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "scratch-resolve-blocked-repo-bin",
		Mode: ReviewProbeScratchOnly,
		Files: []ReviewProbeFile{
			{Path: "check.txt", Content: "ok\n"},
		},
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"check.txt"}},
		},
	})

	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
}
