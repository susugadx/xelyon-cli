package review

var blockedRGHostReadOnlyFlags = []string{
	"--pre",
	"--pre-glob",
}

func validateAndPrepareRGHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedRGHostReadOnlyFlags) {
			return hostReadOnlyCommandPolicyResult{}, newBlockedCommandArgError("rg", arg)
		}
	}
	return newHostReadOnlyPolicyResult(collectSearchCommandPathCandidates("rg", args)), nil
}
