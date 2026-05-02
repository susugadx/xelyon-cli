package review

import (
	"fmt"
	"strings"
)

var blockedFindFlags = []string{
	"-delete",
	"-exec",
	"-execdir",
	"-ok",
	"-okdir",
	"-fprint",
	"-fprint0",
	"-fprintf",
	"-fls",
}

var blockedFindLeadingGlobalOptions = []string{
	"-H",
	"-L",
	"-P",
}

type parsedFindHostReadOnlyArgs struct {
	pathRoots []string
}

func validateAndPrepareFindHostReadOnlyArgs(args []string) (hostReadOnlyCommandAnalysis, error) {
	parsed, err := parseFindHostReadOnlyArgs(args)
	if err != nil {
		return nil, err
	}

	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedFindFlags) {
			return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: find argument %s is not allowed in host_readonly", arg))
		}
	}
	return findHostReadOnlyAnalysis{
		parsed: parsed,
	}, nil
}

func parseFindHostReadOnlyArgs(args []string) (parsedFindHostReadOnlyArgs, error) {
	if len(args) == 0 {
		return parsedFindHostReadOnlyArgs{pathRoots: []string{"."}}, nil
	}

	start := 0
	if args[0] == "--" {
		start = 1
	}

	pathRoots := make([]string, 0, len(args)-start)
	for i := start; i < len(args); i++ {
		arg := args[i]

		if isFindExpressionStartArg(arg) {
			if isBlockedFindLeadingGlobalOption(arg) {
				return parsedFindHostReadOnlyArgs{}, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: find leading option %s is not allowed in host_readonly", arg))
			}
			break
		}

		pathRoots = append(pathRoots, arg)
	}

	if len(pathRoots) == 0 {
		pathRoots = append(pathRoots, ".")
	}

	return parsedFindHostReadOnlyArgs{
		pathRoots: pathRoots,
	}, nil
}

func isFindExpressionStartArg(arg string) bool {
	return strings.HasPrefix(arg, "-") || arg == "!" || arg == "("
}

func isBlockedFindLeadingGlobalOption(arg string) bool {
	if arg == "-D" || strings.HasPrefix(arg, "-D") {
		return true
	}
	if strings.HasPrefix(arg, "-O") {
		return true
	}
	for _, option := range blockedFindLeadingGlobalOptions {
		if arg == option {
			return true
		}
	}
	return false
}
