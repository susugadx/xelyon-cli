package review

// ReviewProbeMode は ProbeRunner/report が共有する probe 実行境界を表す。
// LLM の plan schema 専用 enum ではなく、実際の filesystem/command policy は
// 各 mode runner の validation と実行 policy が扱う。
type ReviewProbeMode string

const (
	// ReviewProbeHostReadOnly は元 repo を read-only bind した process sandbox で実行する。
	ReviewProbeHostReadOnly ReviewProbeMode = "host_readonly"
	// ReviewProbeScratchOnly は repo 外 scratch だけを書き込み可能にした process sandbox で実行する。
	ReviewProbeScratchOnly ReviewProbeMode = "scratch_only"
	// ReviewProbeRepoSandbox は元 repo の現在状態を一時 worktree へコピーし、copy 側だけを bind して実行する。
	ReviewProbeRepoSandbox ReviewProbeMode = "repo_sandbox"
)

func isKnownReviewProbeMode(mode ReviewProbeMode) bool {
	switch mode {
	case ReviewProbeHostReadOnly, ReviewProbeScratchOnly, ReviewProbeRepoSandbox:
		return true
	default:
		return false
	}
}
