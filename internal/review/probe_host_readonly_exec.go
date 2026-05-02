package review

import (
	"context"
	"time"
)

func executeHostReadOnlyCommand(ctx context.Context, cmd hostReadOnlyCommand, timeout time.Duration, maxOutputBytes int64) ReviewProbeCommandResult {
	return executeProbeCommand(ctx, probeExecCommand{
		command: cmd.command,
		args:    cmd.args,
		workDir: cmd.workDir,
	}, timeout, maxOutputBytes)
}
