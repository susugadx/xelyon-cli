package review

import (
	"fmt"
	"strings"
)

const catStdinArg = "-"

func validateGrepHostReadOnlyArgs(_ []string) error {
	return nil
}

func validateLSHostReadOnlyArgs(_ []string) error {
	return nil
}

func validateCatHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: cat requires at least one path argument in host_readonly")
	}
	for _, arg := range args {
		if arg == catStdinArg {
			return fmt.Errorf("blocked command: cat argument %s is not allowed in host_readonly", arg)
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Errorf("blocked command: cat option %s is not allowed in host_readonly", arg)
		}
	}
	return nil
}
