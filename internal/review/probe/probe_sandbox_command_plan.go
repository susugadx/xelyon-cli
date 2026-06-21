package probe

import (
	"path/filepath"
	"strings"
)

type probePlannedCommand struct {
	command     string
	commandPath string
	args        []string
	workDir     string
}

type probeCommandPlanSpec struct {
	workDirMessageLabel string
	rootDir             string
	rootLabel           string
	defaultWorkDir      string
	repoRoot            string
	resolverScratchDir  string
	commandEnv          []string
	analyzePathArgs     func(command string, args []string) ([]string, error)
	validateResolved    func(rootDir, resolvedPath, label string) error
}

func buildProbeCommandPlan(spec probeCommandPlanSpec, cmd ReviewProbeCommand) (probePlannedCommand, error) {
	commandName := strings.TrimSpace(cmd.Command)
	if commandName == "" {
		return probePlannedCommand{}, newBlockedCommandErrorf("command is empty")
	}

	workDir, err := resolveProbeCommandPlanWorkDir(spec, cmd.WorkDir)
	if err != nil {
		return probePlannedCommand{}, err
	}

	pathArgs, err := spec.analyzePathArgs(commandName, cmd.Args)
	if err != nil {
		return probePlannedCommand{}, err
	}
	if err := validateProbeCommandPlanPathArgs(spec, workDir, commandName, pathArgs); err != nil {
		return probePlannedCommand{}, err
	}

	commandPath, err := ResolveCommandPath(commandName, CommandResolutionContext{
		RepoRoot:   spec.repoRoot,
		ScratchDir: spec.resolverScratchDir,
		WorkDir:    workDir,
		Env:        spec.commandEnv,
	})
	if err != nil {
		return probePlannedCommand{}, err
	}

	return probePlannedCommand{
		command:     commandName,
		commandPath: commandPath,
		args:        append([]string(nil), cmd.Args...),
		workDir:     workDir,
	}, nil
}

func resolveProbeCommandPlanWorkDir(spec probeCommandPlanSpec, workDir string) (string, error) {
	trimmed := strings.TrimSpace(workDir)
	if trimmed == "" {
		return spec.defaultWorkDir, nil
	}
	if filepath.IsAbs(trimmed) {
		return "", newBlockedCommandErrorf("%s workdir %q must be relative", spec.workDirMessageLabel, workDir)
	}

	resolved, err := resolvePathWithinRepoRootWithSymlinkCheck(spec.rootDir, spec.defaultWorkDir, trimmed)
	if err != nil {
		if isOutsideRepoPathError(err) {
			return "", newBlockedCommandErrorf("%s workdir %q escapes %s", spec.workDirMessageLabel, workDir, spec.rootLabel)
		}
		return "", newBlockedCommandErrorf("%s workdir %q is invalid: %v", spec.workDirMessageLabel, workDir, err)
	}
	return resolved, nil
}

func validateProbeCommandPlanPathArgs(spec probeCommandPlanSpec, workDir, command string, pathArgs []string) error {
	for _, pathArg := range pathArgs {
		if filepath.IsAbs(pathArg) {
			return newBlockedCommandErrorf("%s path %q is outside %s", command, pathArg, spec.rootLabel)
		}
		resolved, err := resolvePathWithinRepoRootWithSymlinkCheck(spec.rootDir, workDir, pathArg)
		if err != nil {
			if isOutsideRepoPathError(err) {
				return newBlockedCommandErrorf("%s path %q is outside %s", command, pathArg, spec.rootLabel)
			}
			return newBlockedCommandErrorf("failed to resolve %s path %q: %v", command, pathArg, err)
		}
		if spec.validateResolved == nil {
			continue
		}
		if err := spec.validateResolved(spec.rootDir, resolved, command+" path "+pathArg); err != nil {
			return err
		}
	}
	return nil
}
