package review

import (
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
	pathRoots          []string
	firstExpressionArg string
}

func validateAndPrepareFindHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	parsed, err := parseFindHostReadOnlyArgs(args)
	if err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
	}
	if err := validateFindLeadingGlobalOptions(parsed); err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
	}
	if err := validateBlockedFindActionFlags(args); err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
	}
	return newHostReadOnlyPolicyResult(parsed.pathRoots), nil
}

func validateBlockedFindActionFlags(args []string) error {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedFindFlags) {
			return newBlockedCommandArgError("find", arg)
		}
	}
	return nil
}

func validateFindLeadingGlobalOptions(parsed parsedFindHostReadOnlyArgs) error {
	if parsed.firstExpressionArg == "" {
		return nil
	}
	if isBlockedFindLeadingGlobalOption(parsed.firstExpressionArg) {
		return newBlockedCommandErrorf("find leading option %s is not allowed in host_readonly", parsed.firstExpressionArg)
	}
	return nil
}

func parseFindHostReadOnlyArgs(args []string) (parsedFindHostReadOnlyArgs, error) {
	if len(args) == 0 {
		return parsedFindHostReadOnlyArgs{pathRoots: []string{"."}}, nil
	}

	start, treatFirstExpressionAsPathRoot := resolveFindParseStart(args)
	pathRoots, firstExpressionArg := collectFindPathRootsAndExpression(args[start:], treatFirstExpressionAsPathRoot)

	if len(pathRoots) == 0 {
		pathRoots = append(pathRoots, ".")
	}

	return parsedFindHostReadOnlyArgs{
		pathRoots:          pathRoots,
		firstExpressionArg: firstExpressionArg,
	}, nil
}

func resolveFindParseStart(args []string) (start int, treatFirstExpressionAsPathRoot bool) {
	if len(args) == 0 || args[0] != "--" {
		return 0, false
	}
	if len(args) > 1 && isFindExpressionStartArg(args[1]) {
		// `find -- -name ...` のように `--` 直後が式開始トークン風なら、
		// 先頭1つは path root として解釈して outside path 検証対象に残す。
		return 1, true
	}
	return 1, false
}

func collectFindPathRootsAndExpression(args []string, treatFirstExpressionAsPathRoot bool) (pathRoots []string, firstExpressionArg string) {
	pathRoots = make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if isFindExpressionStartArg(arg) {
			if treatFirstExpressionAsPathRoot && len(pathRoots) == 0 {
				pathRoots = append(pathRoots, arg)
				continue
			}
			return pathRoots, arg
		}
		pathRoots = append(pathRoots, arg)
	}

	return pathRoots, ""
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
