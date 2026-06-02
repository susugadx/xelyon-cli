package probe

import "errors"

func validateHostReadOnlyCommandPathPolicy(repoRoot, workDir, command string, args []string) error {
	_, err := planHostReadOnlyCommand(repoRoot, workDir, command, args)
	return err
}

func planHostReadOnlyCommand(repoRoot, workDir, command string, args []string) (analyzedHostReadOnlyCommand, error) {
	analyzed, err := analyzeHostReadOnlyCommand(command, args)
	if err != nil {
		return analyzedHostReadOnlyCommand{}, err
	}
	if err := validateHostReadOnlyCommandPathArgs(repoRoot, workDir, command, analyzed.pathArgs); err != nil {
		return analyzedHostReadOnlyCommand{}, err
	}
	return analyzed, nil
}

func validateHostReadOnlyCommandPathArgs(repoRoot, workDir, command string, pathArgs []string) error {
	for _, pathArg := range pathArgs {
		if err := validateHostReadOnlyPathArgWithinRepo(repoRoot, workDir, command, pathArg); err != nil {
			return err
		}
	}
	return nil
}

func validateHostReadOnlyPathArgWithinRepo(repoRoot, workDir, command, pathArg string) error {
	if _, err := resolvePathWithinRepoRootWithSymlinkCheck(repoRoot, workDir, pathArg); err != nil {
		if errors.Is(err, ErrHostReadOnlyOutsideRepoPath) {
			return newOutsideRepoCommandPathError(command, pathArg)
		}
		return newBlockedCommandErrorf("failed to resolve %s path %q: %v", command, pathArg, err)
	}
	return nil
}
