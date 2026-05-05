package review

// ReviewProbeMode は ProbeRunner/report が共有する probe 実行境界を表す。
// LLM の plan schema 専用 enum ではなく、実際の filesystem/command policy は
// 各 mode runner の validation と実行 policy が扱う。
type ReviewProbeMode string

const (
	// ReviewProbeHostReadOnly は元 repo 上で read-only policy のコマンドだけを実行する。
	ReviewProbeHostReadOnly ReviewProbeMode = "host_readonly"
	// ReviewProbeScratchOnly は repo 外 scratch に生成ファイルだけを置いて実行する。
	ReviewProbeScratchOnly ReviewProbeMode = "scratch_only"
	// ReviewProbeRepoSandbox は元 repo の現在状態を一時 worktree へコピーして実行する。
	// OS/network sandbox ではないため、元 repo の mutation は実行前後 snapshot で検出する。
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
