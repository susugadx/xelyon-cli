package review

import (
	"strings"
)

const catStdinArg = "-"

func validateAndPrepareGrepHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	for _, arg := range args {
		if isBlockedGrepSymlinkTraversalArg(arg) {
			return hostReadOnlyCommandPolicyResult{}, newBlockedCommandArgError("grep", arg)
		}
	}
	return newHostReadOnlyPolicyResult(collectSearchCommandPathCandidates("grep", args)), nil
}

func validateAndPrepareLSHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return newHostReadOnlyNoPathPolicyResult(), nil
	}

	paths := make([]string, 0, len(args))
	optionsTerminated := false
	for _, arg := range args {
		if !optionsTerminated && arg == "--" {
			optionsTerminated = true
			continue
		}
		if !optionsTerminated && isBlockedLSSymlinkDereferenceArg(arg) {
			return hostReadOnlyCommandPolicyResult{}, newBlockedCommandArgError("ls", arg)
		}
		if !optionsTerminated && strings.HasPrefix(arg, "-") {
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

func isBlockedGrepSymlinkTraversalArg(arg string) bool {
	return arg == "--dereference-recursive" ||
		strings.HasPrefix(arg, "--dereference-recursive=") ||
		hasShortOptionFlag(arg, 'R')
}

func isBlockedLSSymlinkDereferenceArg(arg string) bool {
	switch {
	case arg == "--dereference",
		strings.HasPrefix(arg, "--dereference="),
		arg == "--dereference-command-line",
		strings.HasPrefix(arg, "--dereference-command-line="),
		arg == "--dereference-command-line-symlink-to-dir",
		strings.HasPrefix(arg, "--dereference-command-line-symlink-to-dir="):
		return true
	default:
		return hasShortOptionFlag(arg, 'L') || hasShortOptionFlag(arg, 'H')
	}
}

func hasShortOptionFlag(arg string, flag byte) bool {
	if len(arg) < 2 || arg[0] != '-' || arg[1] == '-' {
		return false
	}
	for i := 1; i < len(arg); i++ {
		if arg[i] == flag {
			return true
		}
	}
	return false
}
