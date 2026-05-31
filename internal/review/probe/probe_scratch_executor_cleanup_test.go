package probe

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScratchOnlyExecutor_BlocksScratchDirInsideRepoAndCleansUp(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	scratchDir := filepath.Join(repo, ".xelyon-review-scratch")
	removed := make([]string, 0, 1)

	executor := newScratchOnlyExecutor(repo)
	executor.mktemp = func(dir, pattern string) (string, error) {
		if err := os.MkdirAll(scratchDir, 0o755); err != nil {
			return "", err
		}
		return scratchDir, nil
	}
	executor.removeAll = func(path string) error {
		removed = append(removed, path)
		return os.RemoveAll(path)
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "scratch-inside-repo",
		Mode: ReviewProbeScratchOnly,
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
	if !strings.Contains(result.Error, "outside repository root") {
		t.Fatalf("Error = %q, want to contain outside repository root", result.Error)
	}
	if len(removed) != 1 {
		t.Fatalf("len(removed) = %d, want 1", len(removed))
	}
	if filepath.Clean(removed[0]) != filepath.Clean(scratchDir) {
		t.Fatalf("removed[0] = %q, want %q", removed[0], scratchDir)
	}
	if _, err := os.Stat(scratchDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("scratch dir should be removed, stat error = %v", err)
	}
}

func TestScratchOnlyExecutor_AppendsCleanupErrorOnPassedResult(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	scratchDir := filepath.Join(t.TempDir(), "xelyon-review-scratch-passed")
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(scratchDir)
	})

	executor := newScratchOnlyExecutor(repo)
	executor.mktemp = func(dir, pattern string) (string, error) {
		return scratchDir, nil
	}
	executor.removeAll = func(path string) error {
		return errors.New("cleanup failed")
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "scratch-cleanup-error-passed",
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
	if !strings.Contains(result.Error, "failed to remove scratch directory") {
		t.Fatalf("Error = %q, want to contain cleanup error", result.Error)
	}
}

func TestScratchOnlyExecutor_AppendsCleanupErrorOnBlockedResult(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	scratchDir := filepath.Join(repo, ".xelyon-review-scratch-blocked-cleanup")
	if err := os.MkdirAll(scratchDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(scratchDir)
	})

	executor := newScratchOnlyExecutor(repo)
	executor.mktemp = func(dir, pattern string) (string, error) {
		return scratchDir, nil
	}
	executor.removeAll = func(path string) error {
		return errors.New("cleanup failed")
	}

	result := executor.run(context.Background(), ReviewProbeRequest{
		ID:   "scratch-cleanup-error-blocked",
		Mode: ReviewProbeScratchOnly,
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
	if !strings.Contains(result.Error, "outside repository root") {
		t.Fatalf("Error = %q, want to contain blocked reason", result.Error)
	}
	if !strings.Contains(result.Error, "failed to remove scratch directory") {
		t.Fatalf("Error = %q, want to contain cleanup error", result.Error)
	}
}
