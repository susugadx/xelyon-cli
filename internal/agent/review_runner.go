package agent

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/review"
	"github.com/susugadx/xelyon-cli/internal/reviewadapter"
)

// RunReview は Agent runtime を使って /review current_changes runner を実行する。
func (a *Agent) RunReview(ctx context.Context, req review.ReviewRequest) (review.ReviewReport, error) {
	if a == nil {
		return review.ReviewReport{}, fmt.Errorf("review run: agent is nil")
	}
	ctx, cleanup := a.beginReviewRequestContext(ctx)
	defer cleanup()

	cwd, err := os.Getwd()
	if err != nil {
		return review.ReviewReport{}, fmt.Errorf("review run resolve cwd: %w", err)
	}
	repoRoot, err := resolveReviewRepoRoot(ctx, cwd)
	if err != nil {
		return review.ReviewReport{}, err
	}

	factory := reviewadapter.NewRunnerFactory(reviewadapter.RunnerFactoryOptions{
		RepoRoot: repoRoot,
		CWD:      cwd,
		Model:    agentReviewModel{agent: a},
	})
	runner, err := factory.NewReviewRunner()
	if err != nil {
		return review.ReviewReport{}, fmt.Errorf("review run build runner: %w", err)
	}
	return runner.Run(ctx, req)
}

func resolveReviewRepoRoot(ctx context.Context, cwd string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", cwd, "rev-parse", "--show-toplevel")
	out, err := cmd.CombinedOutput()
	if err != nil {
		detail := strings.TrimSpace(string(out))
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("review run resolve repo root: %s", detail)
	}
	repoRoot := strings.TrimSpace(string(out))
	if repoRoot == "" {
		return "", fmt.Errorf("review run resolve repo root: git returned empty repo root")
	}
	return repoRoot, nil
}
