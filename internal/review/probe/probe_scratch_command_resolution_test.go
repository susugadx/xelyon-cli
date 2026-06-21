package probe

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
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
		Mode: domain.ReviewProbeScratchOnly,
		Files: []ReviewProbeFile{
			{Path: "check.txt", Content: "ok\n"},
		},
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"check.txt"}},
		},
	})

	assertCommandResolutionPassed(t, result, "safe-cat")
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
		Mode: domain.ReviewProbeScratchOnly,
		Files: []ReviewProbeFile{
			{Path: "check.txt", Content: "ok\n"},
		},
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"check.txt"}},
		},
	})

	assertCommandResolutionBlocked(t, result)
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
		Mode: domain.ReviewProbeScratchOnly,
		Files: []ReviewProbeFile{
			{Path: "check.txt", Content: "ok\n"},
		},
		Commands: []ReviewProbeCommand{
			{Command: "cat", Args: []string{"check.txt"}},
		},
	})

	assertCommandResolutionBlocked(t, result)
}
