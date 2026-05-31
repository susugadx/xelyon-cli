package probe

import (
	"errors"
	"fmt"
	"strings"
)

func (e *hostReadOnlyExecutor) validateRequest(req ReviewProbeRequest, commandEnv []string, runtimeRoot string) (hostReadOnlyRequest, error) {
	req = normalizeProbeRequestExecutionLimits(req)

	if req.Mode != ReviewProbeHostReadOnly {
		return hostReadOnlyRequest{}, fmt.Errorf("host_readonly runner received mode %q", req.Mode)
	}
	if len(req.Files) > 0 {
		return hostReadOnlyRequest{}, fmt.Errorf("host_readonly does not allow probe files")
	}
	if len(req.Commands) == 0 {
		return hostReadOnlyRequest{}, fmt.Errorf("probe commands are required")
	}

	commands := make([]hostReadOnlyCommand, 0, len(req.Commands))
	for _, cmd := range req.Commands {
		plannedCommand, err := e.buildHostReadOnlyCommandPlan(commandEnv, runtimeRoot, cmd)
		if err != nil {
			return hostReadOnlyRequest{}, err
		}
		commands = append(commands, plannedCommand)
	}

	return hostReadOnlyRequest{
		id:             req.ID,
		mode:           req.Mode,
		timeout:        req.Timeout,
		maxOutputBytes: req.MaxOutputBytes,
		commands:       commands,
	}, nil
}

func (e *hostReadOnlyExecutor) buildHostReadOnlyCommandPlan(commandEnv []string, runtimeRoot string, cmd ReviewProbeCommand) (hostReadOnlyCommand, error) {
	commandName := strings.TrimSpace(cmd.Command)
	if commandName == "" {
		return hostReadOnlyCommand{}, newBlockedCommandErrorf("command is empty")
	}

	workDir, err := resolveHostReadOnlyWorkDir(e.repoRoot, cmd.WorkDir)
	if err != nil {
		return hostReadOnlyCommand{}, err
	}

	if _, err := planHostReadOnlyCommand(e.repoRoot, workDir, commandName, cmd.Args); err != nil {
		return hostReadOnlyCommand{}, err
	}
	commandPath, err := resolveCommandPath(commandName, commandResolutionContext{
		RepoRoot:   e.repoRoot,
		ScratchDir: runtimeRoot,
		WorkDir:    workDir,
		Env:        commandEnv,
	})
	if err != nil {
		return hostReadOnlyCommand{}, err
	}

	return hostReadOnlyCommand{
		command:     commandName,
		commandPath: commandPath,
		args:        append([]string(nil), cmd.Args...),
		workDir:     workDir,
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
			return "", newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked command: workdir %q is outside repository root", workDir))
		}
		return "", newBlockedCommandErrorf("workdir %q is invalid: %v", workDir, err)
	}
	return resolved, nil
}
