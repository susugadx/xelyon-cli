package review

import "fmt"

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

func validateAndPrepareGoHostReadOnlyArgs(args []string) (hostReadOnlyCommandAnalysis, error) {
	if len(args) == 0 {
		return nil, newHostReadOnlyBlockedError("blocked command: go subcommand is required")
	}
	if _, ok := allowedGoHostReadOnlySubcommands[args[0]]; !ok {
		return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: go %s is not allowed in host_readonly", args[0]))
	}

	for _, arg := range args[1:] {
		if isBlockedFlagArg(arg, blockedGoHostReadOnlyFlags) {
			return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: go argument %s is not allowed in host_readonly", arg))
		}
	}
	return hostReadOnlyNoopAnalysis{}, nil
}
