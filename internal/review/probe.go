package review

import "time"

// ReviewProbeMode は review probe の実行モードを表す。
type ReviewProbeMode string

const (
	ReviewProbeHostReadOnly ReviewProbeMode = "host_readonly"
	ReviewProbeScratchOnly  ReviewProbeMode = "scratch_only"
	ReviewProbeRepoSandbox  ReviewProbeMode = "repo_sandbox"
)

// ReviewProbeStatus は probe 実行結果の状態を表す。
type ReviewProbeStatus string

const (
	ReviewProbePassed          ReviewProbeStatus = "passed"
	ReviewProbeFailed          ReviewProbeStatus = "failed"
	ReviewProbeBlocked         ReviewProbeStatus = "blocked"
	ReviewProbeTimedOut        ReviewProbeStatus = "timed_out"
	ReviewProbeMutatedWorktree ReviewProbeStatus = "mutated_worktree"
)

// ReviewProbeRequest は review 中の検証実行要求を表す。
type ReviewProbeRequest struct {
	ID             string
	Purpose        string
	Mode           ReviewProbeMode
	Files          []ReviewProbeFile
	Commands       []ReviewProbeCommand
	Timeout        time.Duration
	MaxOutputBytes int64
}

// ReviewProbeFile は probe 実行時に必要な一時ファイル定義を表す。
// host_readonly では利用されず、scratch/sandbox 向けの将来拡張として保持する。
type ReviewProbeFile struct {
	Path    string
	Content string
}

// ReviewProbeCommand は probe 内で実行する 1 コマンドを表す。
type ReviewProbeCommand struct {
	Command string
	Args    []string
	WorkDir string
}

// ReviewProbeResult は probe 実行結果を表す。
type ReviewProbeResult struct {
	ID              string
	Mode            ReviewProbeMode
	Status          ReviewProbeStatus
	CommandResults  []ReviewProbeCommandResult
	MutatedWorktree bool
	MutatedFiles    []string
	OutputTruncated bool
	Error           string
}

// ReviewProbeCommandResult は単一コマンドの実行結果を表す。
type ReviewProbeCommandResult struct {
	Command         string
	Args            []string
	WorkDir         string
	Status          ReviewProbeStatus
	ExitCode        int
	Output          string
	OutputTruncated bool
	Error           string
	Duration        time.Duration
}
