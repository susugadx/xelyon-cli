package review

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type scratchOnlyCommand struct {
	command     string
	commandPath string
	args        []string
	workDir     string
}

type scratchOnlyRequest struct {
	id             string
	mode           ReviewProbeMode
	timeout        time.Duration
	maxOutputBytes int64
	files          []scratchOnlyFile
	commands       []scratchOnlyCommand
}

func (e *scratchOnlyExecutor) validateRequest(req ReviewProbeRequest, scratchDir string, commandEnv []string) (scratchOnlyRequest, error) {
	req = normalizeProbeRequestExecutionLimits(req)

	if req.Mode != ReviewProbeScratchOnly {
		return scratchOnlyRequest{}, fmt.Errorf("scratch_only runner received mode %q", req.Mode)
	}
	if len(req.Commands) == 0 {
		return scratchOnlyRequest{}, fmt.Errorf("probe commands are required")
	}
	if len(req.Commands) > defaultScratchOnlyMaxCommands {
		return scratchOnlyRequest{}, newBlockedCommandErrorf("scratch_only allows at most %d commands", defaultScratchOnlyMaxCommands)
	}

	files, err := validateAndBuildScratchFiles(scratchDir, req.Files)
	if err != nil {
		return scratchOnlyRequest{}, err
	}

	commands := make([]scratchOnlyCommand, 0, len(req.Commands))
	for _, cmd := range req.Commands {
		plannedCommand, err := e.buildScratchOnlyCommandPlan(scratchDir, commandEnv, cmd)
		if err != nil {
			return scratchOnlyRequest{}, err
		}
		commands = append(commands, plannedCommand)
	}

	return scratchOnlyRequest{
		id:             req.ID,
		mode:           req.Mode,
		timeout:        req.Timeout,
		maxOutputBytes: req.MaxOutputBytes,
		files:          files,
		commands:       commands,
	}, nil
}

func (e *scratchOnlyExecutor) buildScratchOnlyCommandPlan(scratchDir string, commandEnv []string, cmd ReviewProbeCommand) (scratchOnlyCommand, error) {
	commandName := strings.TrimSpace(cmd.Command)
	if commandName == "" {
		return scratchOnlyCommand{}, newBlockedCommandErrorf("command is empty")
	}

	workDir, err := resolveScratchWorkDir(scratchDir, cmd.WorkDir)
	if err != nil {
		return scratchOnlyCommand{}, err
	}

	pathArgs, err := analyzeScratchOnlyCommand(commandName, cmd.Args)
	if err != nil {
		return scratchOnlyCommand{}, err
	}
	if err := validateScratchPathArgsWithinRoot(scratchDir, workDir, commandName, pathArgs); err != nil {
		return scratchOnlyCommand{}, err
	}
	commandPath, err := resolveProbeCommandPath(commandName, probeCommandResolutionContext{
		RepoRoot:   e.repoRoot,
		ScratchDir: scratchDir,
		WorkDir:    workDir,
		Env:        commandEnv,
	})
	if err != nil {
		return scratchOnlyCommand{}, err
	}

	return scratchOnlyCommand{
		command:     commandName,
		commandPath: commandPath,
		args:        append([]string(nil), cmd.Args...),
		workDir:     workDir,
	}, nil
}

func resolveScratchWorkDir(scratchDir, workDir string) (string, error) {
	trimmed := strings.TrimSpace(workDir)
	if trimmed == "" {
		return scratchDir, nil
	}
	if filepath.IsAbs(trimmed) {
		return "", newBlockedCommandErrorf("scratch command workdir %q must be relative", workDir)
	}

	resolved, err := resolvePathWithinRepoRoot(scratchDir, scratchDir, trimmed)
	if err != nil {
		if isOutsideRepoPathError(err) {
			return "", newBlockedCommandErrorf("scratch command workdir %q escapes scratch directory", workDir)
		}
		return "", newBlockedCommandErrorf("scratch command workdir %q is invalid: %v", workDir, err)
	}
	return resolved, nil
}

func validateScratchPathArgsWithinRoot(scratchDir, workDir, command string, pathArgs []string) error {
	for _, pathArg := range pathArgs {
		if filepath.IsAbs(pathArg) {
			return newBlockedCommandErrorf("%s path %q is outside scratch directory", command, pathArg)
		}
		if _, err := resolvePathWithinRepoRootWithSymlinkCheck(scratchDir, workDir, pathArg); err != nil {
			if isOutsideRepoPathError(err) {
				return newBlockedCommandErrorf("%s path %q is outside scratch directory", command, pathArg)
			}
			return newBlockedCommandErrorf("failed to resolve %s path %q: %v", command, pathArg, err)
		}
	}
	return nil
}
