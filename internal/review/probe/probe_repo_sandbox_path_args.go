package probe

import "strings"

func validateAndCollectRepoSandboxPythonPathArgs(command string, args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, newBlockedCommandErrorf("%s requires a script path in repo_sandbox", command)
	}
	if strings.HasPrefix(args[0], "-") {
		return nil, newBlockedCommandErrorf("%s argument %s is not allowed in repo_sandbox", command, args[0])
	}
	if len(args) > 1 {
		return nil, newBlockedCommandErrorf("%s accepts only one script path in repo_sandbox", command)
	}
	return []string{args[0]}, nil
}

func validateAndCollectRepoSandboxCatPathArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, newBlockedCommandErrorf("cat requires at least one path argument in repo_sandbox")
	}
	for _, arg := range args {
		if arg == catStdinArg {
			return nil, newBlockedCommandErrorf("cat argument %s is not allowed in repo_sandbox", arg)
		}
		if strings.HasPrefix(arg, "-") {
			return nil, newBlockedCommandErrorf("cat option %s is not allowed in repo_sandbox", arg)
		}
	}
	return append([]string(nil), args...), nil
}

func collectRepoSandboxLSPathArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		paths = append(paths, arg)
	}
	return paths
}

func validateAndCollectRepoSandboxFindPathArgs(args []string) ([]string, error) {
	parsed, err := parseFindHostReadOnlyArgs(args)
	if err != nil {
		return nil, err
	}
	if err := validateRepoSandboxFindLeadingGlobalOptions(parsed); err != nil {
		return nil, err
	}
	if err := validateRepoSandboxBlockedFindActionFlags(args); err != nil {
		return nil, err
	}
	return parsed.pathRoots, nil
}

func validateRepoSandboxBlockedFindActionFlags(args []string) error {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedFindFlags) {
			return newBlockedCommandErrorf("find argument %s is not allowed in repo_sandbox", arg)
		}
	}
	return nil
}

func validateRepoSandboxFindLeadingGlobalOptions(parsed parsedFindHostReadOnlyArgs) error {
	if parsed.firstExpressionArg == "" {
		return nil
	}
	if isBlockedFindLeadingGlobalOption(parsed.firstExpressionArg) {
		return newBlockedCommandErrorf("find leading option %s is not allowed in repo_sandbox", parsed.firstExpressionArg)
	}
	return nil
}

func validateAndCollectRepoSandboxRGPathArgs(args []string) ([]string, error) {
	for _, arg := range args {
		if isBlockedRGHostReadOnlyArg(arg) {
			return nil, newBlockedCommandErrorf("rg argument %s is not allowed in repo_sandbox", arg)
		}
	}
	return collectSearchCommandPathCandidates("rg", args), nil
}
