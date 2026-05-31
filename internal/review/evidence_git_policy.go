package review

import reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"

func buildReviewEvidenceGitArgs(repoRoot string, commandArgs []string) []string {
	return reviewprobe.BuildGitArgs(repoRoot, commandArgs)
}

func buildReviewEvidenceGitEnv(environ []string) []string {
	return reviewprobe.BuildGitEnv(environ)
}
