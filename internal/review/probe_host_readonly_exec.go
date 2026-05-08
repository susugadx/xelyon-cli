package review

import (
	"context"
	"time"
)

func executeHostReadOnlyCommand(ctx context.Context, cmd hostReadOnlyCommand, timeout time.Duration, maxOutputBytes int64, env []string, sandbox probeProcessSandbox) ReviewProbeCommandResult {
	return executeProbeCommand(ctx, probeExecCommand{
		command:     cmd.command,
		commandPath: cmd.commandPath,
		args:        cmd.args,
		workDir:     cmd.workDir,
		env:         env,
		sandbox:     sandbox,
	}, timeout, maxOutputBytes)
}
