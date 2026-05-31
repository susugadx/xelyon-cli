package review

import "time"

// ReviewProbeRequest は ProbeRunner.Run に渡す runtime 内部の検証実行要求を表す。
// LLM から直接 decode する schema ではなく、将来の validated ReviewProbePlan から
// 変換された後に runner が扱う契約として維持する。
//
// host_readonly、scratch_only、repo_sandbox の filesystem/command 実行境界は
// この型ではなく、runner 側の validation と policy が扱う。
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
// host_readonly では利用されず、scratch/sandbox 向けの生成ファイルとして扱う。
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
