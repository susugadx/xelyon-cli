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

var (
	// ErrUnsupportedReviewProbeMode は未実装 mode の実行時に返す。
	ErrUnsupportedReviewProbeMode = errors.New("unsupported review probe mode")
)

// ProbeRunner は review probe 実行を担当する。
type ProbeRunner struct {
	repoRoot string
}

// NewProbeRunner は repo root を基準に probe runner を構築する。
func NewProbeRunner(repoRoot string) *ProbeRunner {
	return &ProbeRunner{
		repoRoot: repoRoot,
	}
}

// Run は ReviewProbeRequest を検証してから実行する。
func (r *ProbeRunner) Run(ctx context.Context, req ReviewProbeRequest) (ReviewProbeResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	repoRoot, err := r.resolveRepoRoot()
	if err != nil {
		return ReviewProbeResult{}, err
	}

	switch req.Mode {
	case ReviewProbeHostReadOnly:
		return r.runHostReadOnly(ctx, repoRoot, req), nil
	case ReviewProbeScratchOnly, ReviewProbeRepoSandbox:
		result := ReviewProbeResult{
			ID:     req.ID,
			Mode:   req.Mode,
			Status: ReviewProbeBlocked,
			Error:  fmt.Sprintf("probe mode %q is not implemented yet", req.Mode),
		}
		return result, fmt.Errorf("%w: %s", ErrUnsupportedReviewProbeMode, req.Mode)
	default:
		result := ReviewProbeResult{
			ID:     req.ID,
			Mode:   req.Mode,
			Status: ReviewProbeBlocked,
			Error:  fmt.Sprintf("probe mode %q is not supported", req.Mode),
		}
		return result, fmt.Errorf("%w: %s", ErrUnsupportedReviewProbeMode, req.Mode)
	}
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

func (r *ProbeRunner) runHostReadOnly(ctx context.Context, repoRoot string, req ReviewProbeRequest) ReviewProbeResult {
	result := ReviewProbeResult{
		ID:     req.ID,
		Mode:   req.Mode,
		Status: ReviewProbePassed,
	}

	normalized, err := r.validateHostReadOnlyRequest(repoRoot, req)
	if err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = err.Error()
		return result
	}

	result.ID = normalized.id
	result.Mode = normalized.mode

	beforeSnapshot, err := captureWorktreeSnapshot(ctx, repoRoot)
	if err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = fmt.Sprintf("failed to capture worktree snapshot before probe: %v", err)
		return result
	}

	stop := false
	for _, cmd := range normalized.commands {
		cmdResult := executeHostReadOnlyCommand(ctx, cmd, normalized.timeout, normalized.maxOutputBytes)
		result.CommandResults = append(result.CommandResults, cmdResult)
		result.OutputTruncated = result.OutputTruncated || cmdResult.OutputTruncated

		switch cmdResult.Status {
		case ReviewProbeTimedOut:
			result.Status = ReviewProbeTimedOut
			result.Error = appendError(result.Error, fmt.Sprintf("probe command timed out: %s", formatProbeCommand(cmd.command, cmd.args)))
			stop = true
		case ReviewProbeFailed:
			result.Status = ReviewProbeFailed
			result.Error = appendError(result.Error, fmt.Sprintf("probe command failed: %s", formatProbeCommand(cmd.command, cmd.args)))
			stop = true
		}

		if stop {
			break
		}
	}

	afterSnapshot, err := captureWorktreeSnapshot(ctx, repoRoot)
	if err != nil {
		result.Status = ReviewProbeBlocked
		result.Error = appendError(result.Error, fmt.Sprintf("failed to capture worktree snapshot after probe: %v", err))
		return result
	}

	mutatedFiles := diffWorktreeSnapshots(beforeSnapshot, afterSnapshot)
	if len(mutatedFiles) > 0 {
		result.MutatedWorktree = true
		result.MutatedFiles = mutatedFiles
		result.Status = ReviewProbeMutatedWorktree
		result.Error = appendError(result.Error, "probe command changed the working tree")
	}

	return result
}

func (r *ProbeRunner) validateHostReadOnlyRequest(repoRoot string, req ReviewProbeRequest) (hostReadOnlyRequest, error) {
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

		workDir, err := resolveHostReadOnlyWorkDir(repoRoot, cmd.WorkDir)
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

func validateHostReadOnlyCommandPolicy(command string, args []string) error {
	if strings.ContainsAny(command, `/\`) {
		return fmt.Errorf("blocked command: command path is not allowed in host_readonly: %s", command)
	}

	name := command
	switch name {
	case "git":
		return validateGitHostReadOnlyArgs(args)
	case "rg", "grep", "ls", "find", "cat":
		return nil
	case "sed":
		if len(args) == 0 || args[0] != "-n" {
			return fmt.Errorf("blocked command: sed only supports '-n' in host_readonly")
		}
		return nil
	case "go":
		return validateGoHostReadOnlyArgs(args)
	case "npm":
		return validateNpmHostReadOnlyArgs(args)
	case "cargo":
		return validateCargoHostReadOnlyArgs(args)
	default:
		return fmt.Errorf("blocked command: %s is not allowed in host_readonly", name)
	}
}

func validateGitHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: git subcommand is required")
	}
	switch args[0] {
	case "status", "diff", "show", "grep":
		return nil
	default:
		return fmt.Errorf("blocked command: git %s is not allowed in host_readonly", args[0])
	}
}

func validateGoHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: go subcommand is required")
	}
	switch args[0] {
	case "test", "build", "vet":
		return nil
	default:
		return fmt.Errorf("blocked command: go %s is not allowed in host_readonly", args[0])
	}
}

func validateNpmHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: npm subcommand is required")
	}
	if args[0] == "test" {
		return nil
	}
	if args[0] == "run" && len(args) >= 2 {
		switch args[1] {
		case "test", "lint":
			return nil
		}
	}
	return fmt.Errorf("blocked command: npm %s is not allowed in host_readonly", strings.Join(args, " "))
}

func validateCargoHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: cargo subcommand is required")
	}
	switch args[0] {
	case "test", "clippy":
		return nil
	default:
		return fmt.Errorf("blocked command: cargo %s is not allowed in host_readonly", args[0])
	}
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

func (r *ProbeRunner) resolveRepoRoot() (string, error) {
	root := strings.TrimSpace(r.repoRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to resolve current directory: %w", err)
		}
		root = cwd
	}

	abs, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("failed to resolve repo root %q: %w", root, err)
	}
	return filepath.Clean(abs), nil
}

func formatProbeCommand(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	return command + " " + strings.Join(args, " ")
}

func appendError(base, extra string) string {
	if strings.TrimSpace(extra) == "" {
		return base
	}
	if strings.TrimSpace(base) == "" {
		return extra
	}
	return base + "; " + extra
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
