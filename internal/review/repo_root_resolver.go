package review

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

// ResolveReviewRepoRoot は /review 起動 cwd から review 対象 Git repo root を解決する。
// git 実行境界は EvidenceBuilder と同じ command resolver / env / args policy を使う。
func ResolveReviewRepoRoot(ctx context.Context, cwd string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resolvedCWD, err := reviewevidence.ResolveReviewEvidenceDir(cwd, "")
	if err != nil {
		return "", fmt.Errorf("review run resolve cwd: %w", err)
	}

	lookupRoot := findReviewRepoLookupRoot(resolvedCWD)
	env := reviewevidence.BuildReviewEvidenceGitEnv(os.Environ())
	gitPath, err := reviewprobe.ResolveCommandPath("git", reviewprobe.CommandResolutionContext{
		RepoRoot: lookupRoot,
		WorkDir:  lookupRoot,
		Env:      env,
	})
	if err != nil {
		return "", fmt.Errorf("review run resolve repo root: %w", err)
	}

	result, err := runReviewRepoRootGit(ctx, gitPath, lookupRoot, env)
	if err != nil {
		detail := strings.TrimSpace(result.diagnostics)
		if detail == "" {
			detail = err.Error()
		}
		return "", fmt.Errorf("review run resolve repo root: %s", detail)
	}

	repoRoot := strings.TrimSpace(result.stdout)
	if repoRoot == "" {
		return "", fmt.Errorf("review run resolve repo root: git returned empty repo root")
	}
	return filepath.Clean(repoRoot), nil
}

type reviewRepoRootGitResult struct {
	stdout      string
	diagnostics string
}

func runReviewRepoRootGit(ctx context.Context, gitPath, lookupRoot string, env []string) (reviewRepoRootGitResult, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, gitPath, reviewevidence.BuildReviewEvidenceGitArgs(lookupRoot, []string{"rev-parse", "--show-toplevel"})...)
	cmd.Dir = lookupRoot
	cmd.Env = env
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return reviewRepoRootGitResult{
		stdout:      stdout.String(),
		diagnostics: reviewevidence.CombineReviewEvidenceGitDiagnostics(stderr.String(), stdout.String()),
	}, err
}

func findReviewRepoLookupRoot(cwd string) string {
	current := filepath.Clean(cwd)
	for {
		if hasReviewGitMetadata(current) {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return filepath.Clean(cwd)
		}
		current = parent
	}
}

func hasReviewGitMetadata(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}
