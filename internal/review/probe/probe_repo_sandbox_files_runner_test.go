package probe

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestProbeRunner_RepoSandbox_BlocksWhenGeneratedFileLimitsExceeded(t *testing.T) {
	files := make([]ReviewProbeFile, 0, defaultRepoSandboxMaxGeneratedFiles+1)
	for i := 0; i < defaultRepoSandboxMaxGeneratedFiles+1; i++ {
		files = append(files, ReviewProbeFile{
			Path:    "f-" + strconv.Itoa(i) + ".txt",
			Content: "ok",
		})
	}

	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runner := NewProbeRunner(repo)
	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:       "repo-sandbox-generated-file-limit",
		Mode:     domain.ReviewProbeRepoSandbox,
		Files:    files,
		Commands: []ReviewProbeCommand{{Command: "cat", Args: []string{"keep.txt"}}},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.Status != domain.ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q (error=%q)", result.Status, domain.ReviewProbeBlocked, result.Error)
	}
	if len(result.CommandResults) != 0 {
		t.Fatalf("len(CommandResults) = %d, want 0", len(result.CommandResults))
	}
	if !strings.Contains(result.Error, "allows at most") {
		t.Fatalf("Error = %q, want generated file limit", result.Error)
	}
}
