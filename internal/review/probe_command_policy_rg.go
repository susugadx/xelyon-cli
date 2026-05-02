package review

import "fmt"

var blockedRGHostReadOnlyFlags = []string{
	"--pre",
	"--pre-glob",
}

func validateRGHostReadOnlyArgs(args []string) error {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedRGHostReadOnlyFlags) {
			return fmt.Errorf("blocked command: rg argument %s is not allowed in host_readonly", arg)
		}
	}
	return nil
}
