package review

import (
	"errors"
	"fmt"
	"strings"
)

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
		plannedCommand, err := e.buildHostReadOnlyCommandPlan(cmd)
		if err != nil {
			return hostReadOnlyRequest{}, err
		}
		commands = append(commands, plannedCommand)
	}

	return hostReadOnlyRequest{
		id:             req.ID,
		mode:           req.Mode,
		timeout:        timeout,
		maxOutputBytes: maxOutput,
		commands:       commands,
	}, nil
}

func (e *hostReadOnlyExecutor) buildHostReadOnlyCommandPlan(cmd ReviewProbeCommand) (hostReadOnlyCommand, error) {
	commandName := strings.TrimSpace(cmd.Command)
	if commandName == "" {
		return hostReadOnlyCommand{}, newHostReadOnlyBlockedError("blocked command: command is empty")
	}

	workDir, err := resolveHostReadOnlyWorkDir(e.repoRoot, cmd.WorkDir)
	if err != nil {
		return hostReadOnlyCommand{}, err
	}

	analyzed, err := analyzeHostReadOnlyCommand(commandName, cmd.Args)
	if err != nil {
		return hostReadOnlyCommand{}, err
	}
	if err := validateHostReadOnlyCommandPathArgs(e.repoRoot, workDir, commandName, analyzed.pathArgs); err != nil {
		return hostReadOnlyCommand{}, err
	}

	return hostReadOnlyCommand{
		command: commandName,
		args:    append([]string(nil), cmd.Args...),
		workDir: workDir,
	}, nil
}

func resolveHostReadOnlyWorkDir(repoRoot, workDir string) (string, error) {
	trimmed := strings.TrimSpace(workDir)
	if trimmed == "" {
		return repoRoot, nil
	}

	resolved, err := resolvePathWithinRepoRoot(repoRoot, repoRoot, trimmed)
	if err != nil {
		if errors.Is(err, ErrHostReadOnlyOutsideRepoPath) {
			return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked workdir %q: outside repository root", workDir))
		}
		return "", newHostReadOnlyBlockedError(fmt.Sprintf("blocked workdir %q: %v", workDir, err))
	}
	return resolved, nil
}
