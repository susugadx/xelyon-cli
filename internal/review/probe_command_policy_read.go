package review

import (
	"strings"
)

const catStdinArg = "-"

func validateAndPrepareGrepHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	return newHostReadOnlyPolicyResult(collectSearchCommandPathCandidates("grep", args)), nil
}

func validateAndPrepareLSHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return newHostReadOnlyNoPathPolicyResult(), nil
	}

	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		paths = append(paths, arg)
	}
	return newHostReadOnlyPolicyResult(paths), nil
}

func validateAndPrepareCatHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("cat requires at least one path argument in host_readonly")
	}
	for _, arg := range args {
		if arg == catStdinArg {
			return hostReadOnlyCommandPolicyResult{}, newBlockedCommandArgError("cat", arg)
		}
		if strings.HasPrefix(arg, "-") {
			return hostReadOnlyCommandPolicyResult{}, newBlockedCommandOptionError("cat", arg)
		}
	}
	return newHostReadOnlyPolicyResult(args), nil
}
