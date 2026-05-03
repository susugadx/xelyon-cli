package review

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"
)

type probeExecCommand struct {
	command     string
	commandPath string
	args        []string
	workDir     string
	env         []string
}

func executeProbeCommand(ctx context.Context, cmd probeExecCommand, timeout time.Duration, maxOutputBytes int64) ReviewProbeCommandResult {
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

	execPath := cmd.commandPath
	if strings.TrimSpace(execPath) == "" {
		execPath = cmd.command
	}

	// shell は通さない。mutation につながる tool 固有 args は各 mode の policy で防ぐ。
	proc := exec.CommandContext(cmdCtx, execPath, cmd.args...)
	proc.Dir = cmd.workDir
	if len(cmd.env) > 0 {
		proc.Env = append([]string(nil), cmd.env...)
	}

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
