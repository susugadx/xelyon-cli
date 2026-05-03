package review

import (
	"fmt"
	"time"
)

type scratchOnlyCommand = probePlannedCommand

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
	return buildProbeCommandPlan(probeCommandPlanSpec{
		workDirMessageLabel: "scratch command",
		rootDir:             scratchDir,
		rootLabel:           "scratch directory",
		defaultWorkDir:      scratchDir,
		repoRoot:            e.repoRoot,
		resolverScratchDir:  scratchDir,
		commandEnv:          commandEnv,
		analyzePathArgs:     analyzeScratchOnlyCommand,
		validateResolved: func(rootDir, resolvedPath, label string) error {
			return validateScratchExistingAncestorsWithinRoot(rootDir, resolvedPath, label)
		},
	}, cmd)
}
