package evidence

import reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"

func buildReviewEvidenceGitArgs(repoRoot string, commandArgs []string) []string {
	return reviewprobe.BuildGitArgs(repoRoot, commandArgs)
}

// BuildReviewEvidenceGitArgs は evidence と同じ Git invocation policy の引数を組み立てる。
func BuildReviewEvidenceGitArgs(repoRoot string, commandArgs []string) []string {
	return buildReviewEvidenceGitArgs(repoRoot, commandArgs)
}

func buildReviewEvidenceGitEnv(environ []string) []string {
	return reviewprobe.BuildGitEnv(environ)
}

// BuildReviewEvidenceGitEnv は evidence と同じ Git 実行環境 policy を適用する。
func BuildReviewEvidenceGitEnv(environ []string) []string {
	return buildReviewEvidenceGitEnv(environ)
}
