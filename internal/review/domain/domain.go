package domain

// TargetKind は review 対象の種類を表す。
type TargetKind string

const (
	// TargetCurrentChanges は現在の作業ツリー差分を review 対象にする。
	TargetCurrentChanges TargetKind = "current_changes"
)

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

// IsKnownReviewProbeMode は既知の probe mode かを返す。
func IsKnownReviewProbeMode(mode ReviewProbeMode) bool {
	switch mode {
	case ReviewProbeHostReadOnly, ReviewProbeScratchOnly, ReviewProbeRepoSandbox:
		return true
	default:
		return false
	}
}

// ReviewProbeStatus は probe 実行結果の状態を表す。
type ReviewProbeStatus string

const (
	// ReviewProbePassed は probe が成功した状態を表す。
	ReviewProbePassed ReviewProbeStatus = "passed"
	// ReviewProbeFailed は probe が失敗した状態を表す。
	ReviewProbeFailed ReviewProbeStatus = "failed"
	// ReviewProbeBlocked は probe が実行前または実行中に block された状態を表す。
	ReviewProbeBlocked ReviewProbeStatus = "blocked"
	// ReviewProbeTimedOut は probe が timeout した状態を表す。
	ReviewProbeTimedOut ReviewProbeStatus = "timed_out"
	// ReviewProbeMutatedWorktree は probe が worktree mutation を検出した状態を表す。
	ReviewProbeMutatedWorktree ReviewProbeStatus = "mutated_worktree"
)

// IsKnownReviewProbeStatus は既知の probe status かを返す。
func IsKnownReviewProbeStatus(status ReviewProbeStatus) bool {
	switch status {
	case ReviewProbePassed, ReviewProbeFailed, ReviewProbeBlocked, ReviewProbeTimedOut, ReviewProbeMutatedWorktree:
		return true
	default:
		return false
	}
}
