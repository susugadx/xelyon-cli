package review

import (
	"errors"
)

func validateHostReadOnlyCommandPathPolicy(repoRoot, workDir, command string, args []string) error {
	_, err := analyzeAndValidateHostReadOnlyCommandPaths(repoRoot, workDir, command, args)
	return err
}

func analyzeAndValidateHostReadOnlyCommandPaths(repoRoot, workDir, command string, args []string) ([]string, error) {
	analyzed, err := analyzeHostReadOnlyCommand(command, args)
	if err != nil {
		return nil, err
	}
	if err := validateHostReadOnlyCommandPathArgs(repoRoot, workDir, command, analyzed.pathArgs); err != nil {
		return nil, err
	}
	return analyzed.pathArgs, nil
}

func validateHostReadOnlyCommandPathArgs(repoRoot, workDir, command string, pathArgs []string) error {
	for _, pathArg := range pathArgs {
		if err := validateHostReadOnlyPathArgWithinRepo(repoRoot, workDir, command, pathArg); err != nil {
			return err
		}
	}
	return nil
}

func extractArgsAfterDoubleDash(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			if i+1 >= len(args) {
				return nil
			}
			return append([]string(nil), args[i+1:]...)
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
