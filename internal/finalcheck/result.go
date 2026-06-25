package finalcheck

import (
	"context"
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/turnsupport"
)

const outputTruncateLimit = 2000

const (
	// CommandStatusPassed は final check command が成功した状態を表す。
	CommandStatusPassed = "passed"
	// CommandStatusFailed は final check command が失敗した状態を表す。
	CommandStatusFailed = "failed"
)

// RunResult は final check 実行後に normal turn が必要とする判定結果。
type RunResult struct {
	NeedsContinue      bool
	Feedback           string
	FailureFingerprint string
	Checks             []CommandResult
	Cancelled          bool
	Err                error
}

// CommandResult は final check command の機械可読な実行結果を表す。
type CommandResult struct {
	Command  string
	ExitCode int
	Status   string
}

// BuildFailureResult は失敗した final check command から retry 用 feedback を作る。
func BuildFailureResult(command string, exitCode int, output string, diffStat string) RunResult {
	output = TruncateOutput(output)
	feedback := fmt.Sprintf(`Final check failed. Command %q failed (exit code %d):

%s

[Context] git diff --stat:
%s`, command, exitCode, output, diffStat)

	return RunResult{
		NeedsContinue:      true,
		Feedback:           feedback,
		FailureFingerprint: FailureFingerprint(command, exitCode, output),
	}
}

// BuildCancelledResult は親 context の cancellation による final check 中断結果を作る。
func BuildCancelledResult(err error, checks []CommandResult) RunResult {
	if err == nil {
		err = context.Canceled
	}
	return RunResult{
		Cancelled: true,
		Err:       err,
		Checks:    checks,
	}
}

// TruncateOutput は final check の出力を feedback 用の上限に切り詰める。
func TruncateOutput(output string) string {
	if len(output) > outputTruncateLimit {
		return output[:outputTruncateLimit] + "\n... (truncated)"
	}
	return output
}

// FailureFingerprint は final check 失敗の近似 fingerprint を返す。
func FailureFingerprint(command string, exitCode int, output string) string {
	return turnsupport.ErrorFingerprint(fmt.Sprintf("%s\nexit=%d\n%s", command, exitCode, output))
}
