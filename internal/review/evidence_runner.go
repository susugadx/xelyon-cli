package review

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ReviewEvidenceCommandRunner は EvidenceBuilder が使う git command 実行境界を表す。
type ReviewEvidenceCommandRunner interface {
	RunGit(ctx context.Context, repoRoot, cwd string, args []string, timeout time.Duration, maxOutputBytes int64) (output string, truncated bool, err error)
}

func (b *ReviewEvidenceBuilder) runGit(ctx context.Context, repoRoot, cwd string, args ...string) (string, bool, error) {
	output, truncated, err := b.runner.RunGit(ctx, repoRoot, cwd, args, b.limits.CommandTimeout, b.limits.MaxCommandOutputBytes)
	if err != nil {
		return "", false, err
	}
	return output, truncated, nil
}

type reviewEvidenceGitRunner struct{}

func (reviewEvidenceGitRunner) RunGit(ctx context.Context, repoRoot, cwd string, args []string, timeout time.Duration, maxOutputBytes int64) (string, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	cmdCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	result, err := runReviewEvidenceGitProcess(cmdCtx, reviewEvidenceGitProcessRequest{
		repoRoot:       repoRoot,
		cwd:            cwd,
		args:           args,
		maxOutputBytes: maxOutputBytes,
	})
	if err == nil {
		return result.parsedOutput, result.truncated, nil
	}
	if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
		return result.parsedOutput, result.truncated, fmt.Errorf("git %s timed out: %w", strings.Join(args, " "), cmdCtx.Err())
	}
	return result.parsedOutput, result.truncated, fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(result.diagnostics))
}
