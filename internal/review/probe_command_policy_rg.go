package review

var blockedRGHostReadOnlyFlags = []string{
	"--pre",
	"--pre-glob",
	"--follow",
	"-L",
}

func validateAndPrepareRGHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	for _, arg := range args {
		if isBlockedRGHostReadOnlyArg(arg) {
			return hostReadOnlyCommandPolicyResult{}, newBlockedCommandArgError("rg", arg)
		}
	}
	return newHostReadOnlyPolicyResult(collectSearchCommandPathCandidates("rg", args)), nil
}

func isBlockedRGHostReadOnlyArg(arg string) bool {
	if isBlockedFlagArg(arg, blockedRGHostReadOnlyFlags) {
		return true
	}
	tokens, ok := parseProbeShortOptions(arg, searchCommandPatternSpecs["rg"].shortOptionsWithValue)
	return ok && probeShortOptionsContain(tokens, 'L')
}
