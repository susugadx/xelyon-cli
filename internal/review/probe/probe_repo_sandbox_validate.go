package probe

import (
	"fmt"
	"time"
)

type repoSandboxCommand = probePlannedCommand

type repoSandboxRequest struct {
	id             string
	mode           ReviewProbeMode
	timeout        time.Duration
	maxOutputBytes int64
	files          []repoSandboxFile
	commands       []repoSandboxCommand
}

func (e *repoSandboxExecutor) validateRequest(req ReviewProbeRequest, dirs repoSandboxDirs, commandEnv []string) (repoSandboxRequest, error) {
	req = normalizeProbeRequestExecutionLimits(req)

	if req.Mode != ReviewProbeRepoSandbox {
		return repoSandboxRequest{}, fmt.Errorf("repo_sandbox runner received mode %q", req.Mode)
	}
	if len(req.Commands) == 0 {
		return repoSandboxRequest{}, fmt.Errorf("probe commands are required")
	}
	if len(req.Commands) > defaultRepoSandboxMaxCommands {
		return repoSandboxRequest{}, newBlockedCommandErrorf("repo_sandbox allows at most %d commands", defaultRepoSandboxMaxCommands)
	}

	files, err := validateAndBuildRepoSandboxFiles(dirs.WorktreeDir, req.Files)
	if err != nil {
		return repoSandboxRequest{}, err
	}

	commands := make([]repoSandboxCommand, 0, len(req.Commands))
	for _, cmd := range req.Commands {
		plannedCommand, err := e.buildRepoSandboxCommandPlan(dirs, commandEnv, cmd)
		if err != nil {
			return repoSandboxRequest{}, err
		}
		commands = append(commands, plannedCommand)
	}

	return repoSandboxRequest{
		id:             req.ID,
		mode:           req.Mode,
		timeout:        req.Timeout,
		maxOutputBytes: req.MaxOutputBytes,
		files:          files,
		commands:       commands,
	}, nil
}

func (e *repoSandboxExecutor) buildRepoSandboxCommandPlan(dirs repoSandboxDirs, commandEnv []string, cmd ReviewProbeCommand) (repoSandboxCommand, error) {
	return buildProbeCommandPlan(probeCommandPlanSpec{
		workDirMessageLabel: "repo_sandbox command",
		rootDir:             dirs.WorktreeDir,
		rootLabel:           "sandbox worktree",
		defaultWorkDir:      dirs.WorktreeDir,
		repoRoot:            e.repoRoot,
		resolverScratchDir:  dirs.SandboxRoot,
		commandEnv:          commandEnv,
		analyzePathArgs:     analyzeRepoSandboxCommand,
		validateResolved: func(rootDir, resolvedPath, label string) error {
			return validateRepoSandboxExistingAncestorsWithinWorktree(rootDir, resolvedPath, label)
		},
	}, cmd)
}
