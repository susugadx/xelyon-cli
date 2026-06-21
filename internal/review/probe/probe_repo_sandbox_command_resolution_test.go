package probe

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestRepoSandboxExecutor_CommandResolution(t *testing.T) {
	t.Run("safe external bin allowed", func(t *testing.T) {
		repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
		safeBin := filepath.Join(t.TempDir(), "safe-bin")
		createProbeTestScriptCommand(t, safeBin, "cat", "echo safe-cat")

		executor := newRepoSandboxExecutor(repo)
		executor.baseEnv = []string{"PATH=" + safeBin}

		result := executor.run(context.Background(), ReviewProbeRequest{
			ID:       "repo-sandbox-resolve-safe",
			Mode:     domain.ReviewProbeRepoSandbox,
			Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"keep.txt"}}},
		})
		assertCommandResolutionPassed(t, result, "safe-cat")
	})

	t.Run("original repo bin blocked", func(t *testing.T) {
		repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
		repoBin := filepath.Join(repo, "bin")
		safeBin := filepath.Join(t.TempDir(), "safe-bin")
		createProbeTestScriptCommand(t, repoBin, "cat", "echo repo-cat")
		createProbeTestScriptCommand(t, safeBin, "cat", "echo safe-cat")

		executor := newRepoSandboxExecutor(repo)
		executor.baseEnv = []string{"PATH=" + strings.Join([]string{repoBin, safeBin}, string(filepath.ListSeparator))}

		result := executor.run(context.Background(), ReviewProbeRequest{
			ID:       "repo-sandbox-resolve-blocked-repo-bin",
			Mode:     domain.ReviewProbeRepoSandbox,
			Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"keep.txt"}}},
		})
		assertCommandResolutionBlocked(t, result)
	})

	t.Run("sandbox root bin blocked", func(t *testing.T) {
		repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
		sandboxRoot := filepath.Join(t.TempDir(), "xelyon-review-sandbox-resolve")
		sandboxBin := filepath.Join(sandboxRoot, "bin")
		safeBin := filepath.Join(t.TempDir(), "safe-bin")
		createProbeTestScriptCommand(t, sandboxBin, "cat", "echo sandbox-cat")
		createProbeTestScriptCommand(t, safeBin, "cat", "echo safe-cat")

		executor := newRepoSandboxExecutor(repo)
		executor.mktemp = func(dir, pattern string) (string, error) {
			return sandboxRoot, nil
		}
		executor.baseEnv = []string{"PATH=" + strings.Join([]string{sandboxBin, safeBin}, string(filepath.ListSeparator))}

		result := executor.run(context.Background(), ReviewProbeRequest{
			ID:       "repo-sandbox-resolve-blocked-sandbox-bin",
			Mode:     domain.ReviewProbeRepoSandbox,
			Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"keep.txt"}}},
		})
		assertCommandResolutionBlocked(t, result)
	})
}
