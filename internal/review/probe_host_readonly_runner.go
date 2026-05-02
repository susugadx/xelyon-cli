package review

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultReviewProbeTimeout        = 30 * time.Second
	defaultReviewProbeMaxOutputBytes = 64 * 1024
)

type hostReadOnlyExecutor struct {
	repoRoot string
}

func newHostReadOnlyExecutor(repoRoot string) *hostReadOnlyExecutor {
	return &hostReadOnlyExecutor{repoRoot: repoRoot}
}

type hostReadOnlyCommand struct {
	command string
	args    []string
	workDir string
}

type hostReadOnlyRequest struct {
	id             string
	mode           ReviewProbeMode
	timeout        time.Duration
	maxOutputBytes int64
	commands       []hostReadOnlyCommand
}

func (e *hostReadOnlyExecutor) run(ctx context.Context, req ReviewProbeRequest) ReviewProbeResult {
	result := ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: ReviewProbePassed,
	}

	normalized, err := e.validateRequest(req)
	if err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = err.Error()
		return result
	}

	result.ID = normalized.id
	result.Mode = normalized.mode

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyHostReadOnlySnapshotError(&result, "before", err)
		return result
	}

	for _, cmd := range normalized.commands {
		cmdResult := executeHostReadOnlyCommand(ctx, cmd, normalized.timeout, normalized.maxOutputBytes)
		if applyHostReadOnlyCommandTransition(&result, cmd, cmdResult) {
			break
		}
	}

	afterSnapshot, err := captureWorktreeSnapshot(ctx, e.repoRoot)
	if err != nil {
		applyHostReadOnlySnapshotError(&result, "after", err)
		return result
	}

	applyHostReadOnlyMutationTransition(&result, diffWorktreeSnapshots(beforeSnapshot, afterSnapshot))

	return result
}

func applyHostReadOnlySnapshotError(result *ReviewProbeResult, phase string, err error) {
	result.Status = ReviewProbeBlocked
	result.Error = appendError(result.Error, fmt.Sprintf("failed to capture worktree snapshot %s probe: %v", phase, err))
}

func applyHostReadOnlyCommandTransition(result *ReviewProbeResult, cmd hostReadOnlyCommand, cmdResult ReviewProbeCommandResult) (stop bool) {
	result.CommandResults = append(result.CommandResults, cmdResult)
	result.OutputTruncated = result.OutputTruncated || cmdResult.OutputTruncated

	switch cmdResult.Status {
	case ReviewProbeTimedOut:
		result.Status = ReviewProbeTimedOut
		result.Error = appendError(result.Error, fmt.Sprintf("probe command timed out: %s", formatProbeCommand(cmd.command, cmd.args)))
		return true
	case ReviewProbeFailed:
		result.Status = ReviewProbeFailed
		result.Error = appendError(result.Error, fmt.Sprintf("probe command failed: %s", formatProbeCommand(cmd.command, cmd.args)))
		return true
	default:
		return false
	}
}

func applyHostReadOnlyMutationTransition(result *ReviewProbeResult, mutatedFiles []string) {
	if len(mutatedFiles) == 0 {
		return
	}

	result.MutatedWorktree = true
	result.MutatedFiles = mutatedFiles
	result.Status = ReviewProbeMutatedWorktree
	result.Error = appendError(result.Error, "probe command changed the working tree")
}

func (e *hostReadOnlyExecutor) validateRequest(req ReviewProbeRequest) (hostReadOnlyRequest, error) {
	if req.Mode != ReviewProbeHostReadOnly {
		return hostReadOnlyRequest{}, fmt.Errorf("host_readonly runner received mode %q", req.Mode)
	}
	if len(req.Files) > 0 {
		return hostReadOnlyRequest{}, fmt.Errorf("host_readonly does not allow probe files")
	}
	if len(req.Commands) == 0 {
		return hostReadOnlyRequest{}, fmt.Errorf("probe commands are required")
	}

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = defaultReviewProbeTimeout
	}
	maxOutput := req.MaxOutputBytes
	if maxOutput <= 0 {
		maxOutput = defaultReviewProbeMaxOutputBytes
	}

	commands := make([]hostReadOnlyCommand, 0, len(req.Commands))
	for _, cmd := range req.Commands {
		commandName := strings.TrimSpace(cmd.Command)
		if commandName == "" {
			return hostReadOnlyRequest{}, fmt.Errorf("probe command is empty")
		}
		if err := validateHostReadOnlyCommandPolicy(commandName, cmd.Args); err != nil {
			return hostReadOnlyRequest{}, err
		}

		workDir, err := resolveHostReadOnlyWorkDir(e.repoRoot, cmd.WorkDir)
		if err != nil {
			return hostReadOnlyRequest{}, err
		}

		commands = append(commands, hostReadOnlyCommand{
			command: commandName,
			args:    append([]string(nil), cmd.Args...),
			workDir: workDir,
		})
	}

	return hostReadOnlyRequest{
		id:             req.ID,
		mode:           req.Mode,
		timeout:        timeout,
		maxOutputBytes: maxOutput,
		commands:       commands,
	}, nil
}

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

func resolveHostReadOnlyWorkDir(repoRoot, workDir string) (string, error) {
	base := repoRoot
	if strings.TrimSpace(workDir) == "" {
		return base, nil
	}

	candidate := workDir
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(base, candidate)
	}
	candidate = filepath.Clean(candidate)

	rel, err := filepath.Rel(base, candidate)
	if err != nil {
		return "", fmt.Errorf("blocked workdir %q: %w", workDir, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("blocked workdir %q: outside repository root", workDir)
	}
	return candidate, nil
}

type cappedOutput struct {
	maxBytes int64

	mu        sync.Mutex
	buf       bytes.Buffer
	truncated bool
}

func newCappedOutput(maxBytes int64) *cappedOutput {
	return &cappedOutput{
		maxBytes: maxBytes,
	}
}

func (c *cappedOutput) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.maxBytes <= 0 {
		_, _ = c.buf.Write(p)
		return len(p), nil
	}

	remaining := c.maxBytes - int64(c.buf.Len())
	if remaining <= 0 {
		c.truncated = true
		return len(p), nil
	}

	if int64(len(p)) <= remaining {
		_, _ = c.buf.Write(p)
		return len(p), nil
	}

	_, _ = c.buf.Write(p[:remaining])
	c.truncated = true
	return len(p), nil
}

func (c *cappedOutput) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.String()
}

func (c *cappedOutput) Truncated() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.truncated
}
