package finalcheck

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/turnsupport"
)

const outputTruncateLimit = 2000

// RunResult は final check 実行後に normal turn が必要とする判定結果。
type RunResult struct {
	NeedsContinue      bool
	Feedback           string
	FailureFingerprint string
}

// BuildFailureResult は失敗した final check command から retry 用 feedback を作る。
func BuildFailureResult(command string, exitCode int, output string, diffStat string) RunResult {
	output = TruncateOutput(output)
	feedback := fmt.Sprintf(`[SYSTEM] Final check failed. Command %q failed (exit code %d):

%s

[Context] git diff --stat:
%s

Please fix these errors before declaring completion. Do NOT skip these issues.`, command, exitCode, output, diffStat)

	return RunResult{
		NeedsContinue:      true,
		Feedback:           feedback,
		FailureFingerprint: FailureFingerprint(command, exitCode, output),
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
