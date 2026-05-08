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
	sandbox     probeProcessSandbox
}

func executeProbeCommand(ctx context.Context, cmd probeExecCommand, timeout time.Duration, maxOutputBytes int64) ReviewProbeCommandResult {
	result := ReviewProbeCommandResult{
		Command:  cmd.command,
		Args:     append([]string(nil), cmd.args...),
		WorkDir:  cmd.workDir,
		Status:   ReviewProbePassed,
		ExitCode: -1,
	}

	execPath := strings.TrimSpace(cmd.commandPath)
	if execPath == "" {
		result.Status = ReviewProbeBlocked
		result.Error = probeExecMissingResolvedPathError
		return result
	}

	execCmd, sandboxErr := buildProbeProcessSandboxExec(cmd)
	if sandboxErr != nil {
		result.Status = ReviewProbeBlocked
		result.Error = sandboxErr.Error()
		return result
	}
	execPath = strings.TrimSpace(execCmd.commandPath)
	if execPath == "" {
		result.Status = ReviewProbeBlocked
		result.Error = probeExecMissingResolvedPathError
		return result
	}

	cmdCtx := ctx
	cancel := func() {}
	if timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	// executeProbeCommand は解決済み commandPath を必須とする。shell は通さない。
	// mutation につながる tool 固有 args は各 mode の policy で防ぐ。
	proc := exec.CommandContext(cmdCtx, execPath, execCmd.args...)
	proc.Dir = execCmd.workDir
	if len(execCmd.env) > 0 {
		proc.Env = append([]string(nil), execCmd.env...)
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

	if isProbeSearchNoMatchesExit(cmd.command, cmd.args, result.ExitCode) {
		return result
	}

	result.Status = ReviewProbeFailed
	result.Error = err.Error()
	return result
}

func isProbeSearchNoMatchesExit(command string, args []string, exitCode int) bool {
	if exitCode != 1 {
		return false
	}

	switch command {
	case "rg", "grep":
		return true
	case "git":
		return isProbeGitGrepCommand(args)
	default:
		return false
	}
}

func isProbeGitGrepCommand(args []string) bool {
	parsed, err := parseGitHostReadOnlyArgs(args)
	return err == nil && parsed.subcommand == "grep"
}
