package review

import (
	"fmt"
	"strings"
)

const catStdinArg = "-"

func validateAndPrepareGrepHostReadOnlyArgs(_ []string) (hostReadOnlyCommandAnalysis, error) {
	return hostReadOnlyNoopAnalysis{}, nil
}

func validateAndPrepareLSHostReadOnlyArgs(_ []string) (hostReadOnlyCommandAnalysis, error) {
	return hostReadOnlyNoopAnalysis{}, nil
}

func validateAndPrepareCatHostReadOnlyArgs(args []string) (hostReadOnlyCommandAnalysis, error) {
	if len(args) == 0 {
		return nil, newHostReadOnlyBlockedError("blocked command: cat requires at least one path argument in host_readonly")
	}
	for _, arg := range args {
		if arg == catStdinArg {
			return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: cat argument %s is not allowed in host_readonly", arg))
		}
		if strings.HasPrefix(arg, "-") {
			return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: cat option %s is not allowed in host_readonly", arg))
		}
	}
	return hostReadOnlyNoopAnalysis{}, nil
}
