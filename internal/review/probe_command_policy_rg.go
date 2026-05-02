package review

import "fmt"

var blockedRGHostReadOnlyFlags = []string{
	"--pre",
	"--pre-glob",
}

func validateAndPrepareRGHostReadOnlyArgs(args []string) (hostReadOnlyCommandAnalysis, error) {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedRGHostReadOnlyFlags) {
			return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: rg argument %s is not allowed in host_readonly", arg))
		}
	}
	return hostReadOnlyNoopAnalysis{}, nil
}
