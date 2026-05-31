package review

import "github.com/susugadx/xelyon-cli/internal/review/domain"

// TargetKind は review 対象の種類を表す。
type TargetKind = domain.TargetKind

const (
	// TargetCurrentChanges は現在の作業ツリー差分を review 対象にする。
	TargetCurrentChanges = domain.TargetCurrentChanges
)

// ReviewProbeMode は ProbeRunner/report が共有する probe 実行境界を表す。
type ReviewProbeMode = domain.ReviewProbeMode

const (
	// ReviewProbeHostReadOnly は元 repo を read-only bind した process sandbox で実行する。
	ReviewProbeHostReadOnly = domain.ReviewProbeHostReadOnly
	// ReviewProbeScratchOnly は repo 外 scratch だけを書き込み可能にした process sandbox で実行する。
	ReviewProbeScratchOnly = domain.ReviewProbeScratchOnly
	// ReviewProbeRepoSandbox は元 repo の現在状態を一時 worktree へコピーし、copy 側だけを bind して実行する。
	ReviewProbeRepoSandbox = domain.ReviewProbeRepoSandbox
)

func isKnownReviewProbeMode(mode ReviewProbeMode) bool {
	return domain.IsKnownReviewProbeMode(mode)
}

// ReviewProbeStatus は probe 実行結果の状態を表す。
type ReviewProbeStatus = domain.ReviewProbeStatus

const (
	// ReviewProbePassed は probe が成功した状態を表す。
	ReviewProbePassed = domain.ReviewProbePassed
	// ReviewProbeFailed は probe が失敗した状態を表す。
	ReviewProbeFailed = domain.ReviewProbeFailed
	// ReviewProbeBlocked は probe が実行前または実行中に block された状態を表す。
	ReviewProbeBlocked = domain.ReviewProbeBlocked
	// ReviewProbeTimedOut は probe が timeout した状態を表す。
	ReviewProbeTimedOut = domain.ReviewProbeTimedOut
	// ReviewProbeMutatedWorktree は probe が worktree mutation を検出した状態を表す。
	ReviewProbeMutatedWorktree = domain.ReviewProbeMutatedWorktree
)
