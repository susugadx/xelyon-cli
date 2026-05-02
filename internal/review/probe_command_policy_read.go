package review

import (
	"fmt"
	"strings"
)

const catStdinArg = "-"

func validateAndPrepareGrepHostReadOnlyArgs(_ []string) (hostReadOnlyCommandState, error) {
	return hostReadOnlyCommandState{}, nil
}

func validateAndPrepareLSHostReadOnlyArgs(_ []string) (hostReadOnlyCommandState, error) {
	return hostReadOnlyCommandState{}, nil
}

func validateAndPrepareCatHostReadOnlyArgs(args []string) (hostReadOnlyCommandState, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandState{}, newHostReadOnlyBlockedError("blocked command: cat requires at least one path argument in host_readonly")
	}
	for _, arg := range args {
		if arg == catStdinArg {
			return hostReadOnlyCommandState{}, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: cat argument %s is not allowed in host_readonly", arg))
		}
		if strings.HasPrefix(arg, "-") {
			return hostReadOnlyCommandState{}, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: cat option %s is not allowed in host_readonly", arg))
		}
	}
	return hostReadOnlyCommandState{}, nil
}
