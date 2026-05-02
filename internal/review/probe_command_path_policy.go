package review

import (
	"errors"
	"fmt"
	"strings"
)

func validateHostReadOnlyCommandPathPolicy(repoRoot, workDir, command string, args []string) error {
	analyzed, err := analyzeHostReadOnlyCommand(command, args)
	if err != nil {
		return err
	}
	return validateHostReadOnlyCommandPathArgs(repoRoot, workDir, command, analyzed.pathArgs)
}

func validateHostReadOnlyCommandPathArgs(repoRoot, workDir, command string, pathArgs []string) error {
	for _, pathArg := range pathArgs {
		if err := validateHostReadOnlyPathArgWithinRepo(repoRoot, workDir, command, pathArg); err != nil {
			return err
		}
	}
	return nil
}

func extractCatPathArgsFromCommandArgs(args []string, _ hostReadOnlyCommandAnalysis) ([]string, error) {
	return append([]string(nil), args...), nil
}

func extractLSPathArgsFromCommandArgs(args []string, _ hostReadOnlyCommandAnalysis) ([]string, error) {
	if len(args) == 0 {
		return nil, nil
	}

	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		paths = append(paths, arg)
	}
	return paths, nil
}

func extractFindPathRootsFromCommandArgs(_ []string, analysis hostReadOnlyCommandAnalysis) ([]string, error) {
	findAnalysis, ok := analysis.(findHostReadOnlyAnalysis)
	if !ok {
		return nil, newHostReadOnlyBlockedError("blocked command: internal policy error for find path analysis")
	}
	return append([]string(nil), findAnalysis.parsed.pathRoots...), nil
}

func extractArgsAfterDoubleDashFromCommandArgs(args []string, _ hostReadOnlyCommandAnalysis) ([]string, error) {
	return extractArgsAfterDoubleDash(args), nil
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
			return newHostReadOnlyOutsideRepoPathError(fmt.Sprintf("blocked command: %s path %q is outside repository root", command, pathArg))
		}
		return newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: failed to resolve %s path %q: %v", command, pathArg, err))
	}
	return nil
}
