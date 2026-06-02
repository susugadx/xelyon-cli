package evidence

import (
	"context"
	"os"
	"os/exec"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

type reviewEvidenceGitProcessRequest struct {
	repoRoot       string
	args           []string
	maxOutputBytes int64
}

type reviewEvidenceGitProcessResult struct {
	parsedOutput string
	diagnostics  string
	truncated    bool
}

func runReviewEvidenceGitProcess(ctx context.Context, req reviewEvidenceGitProcessRequest) (reviewEvidenceGitProcessResult, error) {
	proc, err := newReviewEvidenceGitProcess(ctx, req)
	if err != nil {
		return reviewEvidenceGitProcessResult{}, err
	}
	streams := newReviewEvidenceGitProcessStreams(req.maxOutputBytes)
	streams.attach(proc)

	err = proc.Run()
	return streams.result(), err
}

func newReviewEvidenceGitProcess(ctx context.Context, req reviewEvidenceGitProcessRequest) (*exec.Cmd, error) {
	env := buildReviewEvidenceGitEnv(os.Environ())
	gitPath, err := resolveReviewEvidenceGitExecutable(req.repoRoot, env)
	if err != nil {
		return nil, err
	}

	proc := exec.CommandContext(ctx, gitPath, buildReviewEvidenceGitArgs(req.repoRoot, req.args)...)
	proc.Dir = req.repoRoot
	proc.Env = env
	return proc, nil
}

func resolveReviewEvidenceGitExecutable(repoRoot string, env []string) (string, error) {
	return reviewprobe.ResolveCommandPath("git", reviewprobe.CommandResolutionContext{
		RepoRoot: repoRoot,
		WorkDir:  repoRoot,
		Env:      env,
	})
}

type reviewEvidenceGitProcessStreams struct {
	stdout *cappedOutput
	stderr *cappedOutput
}

func newReviewEvidenceGitProcessStreams(maxOutputBytes int64) reviewEvidenceGitProcessStreams {
	return reviewEvidenceGitProcessStreams{
		stdout: newCappedOutput(maxOutputBytes),
		stderr: newCappedOutput(maxOutputBytes),
	}
}

func (s reviewEvidenceGitProcessStreams) attach(proc *exec.Cmd) {
	proc.Stdout = s.stdout
	proc.Stderr = s.stderr
}

func (s reviewEvidenceGitProcessStreams) result() reviewEvidenceGitProcessResult {
	stdout := s.stdout.String()
	stderr := s.stderr.String()
	return reviewEvidenceGitProcessResult{
		parsedOutput: stdout,
		diagnostics:  combineReviewEvidenceGitDiagnostics(stderr, stdout),
		truncated:    s.stdout.Truncated(),
	}
}

func combineReviewEvidenceGitDiagnostics(stderr, stdout string) string {
	if stderr == "" {
		return stdout
	}
	if stdout == "" {
		return stderr
	}
	return stderr + "\n" + stdout
}

// CombineReviewEvidenceGitDiagnostics は evidence と同じ順序で Git diagnostics を結合する。
func CombineReviewEvidenceGitDiagnostics(stderr, stdout string) string {
	return combineReviewEvidenceGitDiagnostics(stderr, stdout)
}
