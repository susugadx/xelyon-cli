package review

import "fmt"

func validateGoHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: go subcommand is required")
	}
	switch args[0] {
	case "test", "build", "vet":
	default:
		return fmt.Errorf("blocked command: go %s is not allowed in host_readonly", args[0])
	}

	for _, arg := range args[1:] {
		if isBlockedFlagArg(arg, []string{
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
		}) {
			return fmt.Errorf("blocked command: go argument %s is not allowed in host_readonly", arg)
		}
	}
	return nil
}
