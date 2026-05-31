package probe

import "strings"

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
	"-follow",
	"-files0-from",
	"-anewer",
	"-cnewer",
	"-samefile",
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

type findParsePhase int

const (
	findParsePhasePathRoots findParsePhase = iota
	findParsePhaseExpression
)

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
		if isBlockedFindFlagArg(arg) {
			return newBlockedCommandArgError("find", arg)
		}
	}
	return nil
}

func isBlockedFindFlagArg(arg string) bool {
	if isBlockedFlagArg(arg, blockedFindFlags) {
		return true
	}
	return strings.HasPrefix(arg, "-newer")
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

	startIndex := 0
	treatFirstExpressionAsPathRoot := false
	if args[0] == "--" {
		startIndex = 1
		if len(args) > 1 && isFindExpressionStartArg(args[1]) {
			// `find -- -name ...` のように `--` 直後が式開始トークン風なら、
			// 先頭1つは path root として解釈して outside path 検証対象に残す。
			treatFirstExpressionAsPathRoot = true
		}
	}

	pathRoots, firstExpressionArg := scanFindPathRoots(args[startIndex:], treatFirstExpressionAsPathRoot)
	if len(pathRoots) == 0 {
		pathRoots = append(pathRoots, ".")
	}

	return parsedFindHostReadOnlyArgs{
		pathRoots:          pathRoots,
		firstExpressionArg: firstExpressionArg,
	}, nil
}

func scanFindPathRoots(args []string, treatFirstExpressionAsPathRoot bool) (pathRoots []string, firstExpressionArg string) {
	pathRoots = make([]string, 0, len(args))
	phase := findParsePhasePathRoots

	for _, arg := range args {
		switch phase {
		case findParsePhasePathRoots:
			if isFindExpressionStartArg(arg) {
				if treatFirstExpressionAsPathRoot && len(pathRoots) == 0 {
					pathRoots = append(pathRoots, arg)
					treatFirstExpressionAsPathRoot = false
					continue
				}
				firstExpressionArg = arg
				phase = findParsePhaseExpression
				continue
			}
			pathRoots = append(pathRoots, arg)
		case findParsePhaseExpression:
			// path root 収集は式開始まで。以降は評価対象外。
			continue
		}
	}

	return pathRoots, firstExpressionArg
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
