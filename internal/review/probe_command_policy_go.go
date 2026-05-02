package review

var (
	allowedGoHostReadOnlySubcommands = map[string]struct{}{
		"test":  {},
		"build": {},
		"vet":   {},
	}
	blockedGoHostReadOnlyFlags = []string{
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
	}
)

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
	return newHostReadOnlyNoPathPolicyResult(), nil
}
