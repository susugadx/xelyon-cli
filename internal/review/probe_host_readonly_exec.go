package review

import (
	"context"
	"errors"
	"os/exec"
	"time"
)

func executeHostReadOnlyCommand(ctx context.Context, cmd hostReadOnlyCommand, timeout time.Duration, maxOutputBytes int64) ReviewProbeCommandResult {
	result := ReviewProbeCommandResult{
		Command:  cmd.command,
		Args:     append([]string(nil), cmd.args...),
		WorkDir:  cmd.workDir,
		Status:   ReviewProbePassed,
		ExitCode: -1,
	}

	cmdCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	// shell は通さない。mutation につながる tool 固有 args は host_readonly policy で防ぐ。
	proc := exec.CommandContext(cmdCtx, cmd.command, cmd.args...)
	proc.Dir = cmd.workDir

	output := newCappedOutput(maxOutputBytes)
	proc.Stdout = output
	proc.Stderr = output

	start := time.Now()
	err := proc.Run()
	result.Duration = time.Since(start)
	result.Output = output.String()
	result.OutputTruncated = output.Truncated()

	if proc.ProcessState != nil {
		result.ExitCode = proc.ProcessState.ExitCode()
	}

	if err == nil {
		return result
	}

	if errors.Is(cmdCtx.Err(), context.DeadlineExceeded) {
		result.Status = ReviewProbeTimedOut
		result.Error = cmdCtx.Err().Error()
		return result
	}

	result.Status = ReviewProbeFailed
	result.Error = err.Error()
	return result
}
