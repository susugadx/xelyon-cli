package review

import "strings"

var (
	allowedGoHostReadOnlySubcommands = map[string]struct{}{
		"test":  {},
		"build": {},
		"vet":   {},
	}
	blockedGoHostReadOnlyFlags = withGoDoubleDashVariants([]string{
		"-c",
		"-coverprofile",
		"-o",
		"-output",
		"-outputdir",
		"-cpuprofile",
		"-memprofile",
		"-mutexprofile",
		"-blockprofile",
		"-trace",
		"-exec",
		"-toolexec",
		"-vettool",
		"-C",
		"-modfile",
		"-overlay",
		"-pkgdir",
	})
	goHostReadOnlyFlagsWithValue = map[string]struct{}{
		"-asmflags":         {},
		"-bench":            {},
		"-benchtime":        {},
		"-blockprofilerate": {},
		"-covermode":        {},
		"-coverpkg":         {},
		"-count":            {},
		"-cpu":              {},
		"-gcflags":          {},
		"-ldflags":          {},
		"-list":             {},
		"-mod":              {},
		"-p":                {},
		"-parallel":         {},
		"-fuzz":             {},
		"-fuzztime":         {},
		"-fuzzminimizetime": {},
		"-run":              {},
		"-shuffle":          {},
		"-skip":             {},
		"-tags":             {},
		"-timeout":          {},
		"-vet":              {},
	}
)

func withGoDoubleDashVariants(flags []string) []string {
	expanded := make([]string, 0, len(flags)*2)
	for _, flag := range flags {
		expanded = append(expanded, flag)
		if strings.HasPrefix(flag, "-") && !strings.HasPrefix(flag, "--") {
			expanded = append(expanded, "-"+flag)
		}
	}
	return expanded
}

func validateAndPrepareGoHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("go subcommand is required")
	}
	if _, ok := allowedGoHostReadOnlySubcommands[args[0]]; !ok {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("go %s is not allowed in host_readonly", args[0])
	}

	for _, arg := range args[1:] {
		if isBlockedFlagArg(arg, blockedGoHostReadOnlyFlags) {
			return hostReadOnlyCommandPolicyResult{}, newBlockedCommandArgError("go", arg)
		}
	}
	return newHostReadOnlyPolicyResult(collectGoHostReadOnlyPathArgs(args[1:])), nil
}

func collectGoHostReadOnlyPathArgs(args []string) []string {
	pathArgs := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-args" {
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			name := goFlagName(arg)
			if goHostReadOnlyFlagConsumesNext(name, arg) && i+1 < len(args) {
				i++
			}
			continue
		}
		if isPathLikeGoOrCargoValue(arg) || strings.HasPrefix(arg, ".") {
			pathArgs = append(pathArgs, arg)
		}
	}
	return pathArgs
}

func goFlagName(arg string) string {
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		return arg[:idx]
	}
	return arg
}

func goHostReadOnlyFlagConsumesNext(name, arg string) bool {
	if strings.Contains(arg, "=") {
		return false
	}
	_, ok := goHostReadOnlyFlagsWithValue[name]
	return ok
}
