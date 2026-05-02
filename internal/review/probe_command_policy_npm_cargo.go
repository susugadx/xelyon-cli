package review

import (
	"fmt"
	"strings"
)

var allowedNpmRunHostReadOnlyScripts = map[string]struct{}{
	"test": {},
	"lint": {},
}

var allowedCargoHostReadOnlySubcommands = map[string]struct{}{
	"test":   {},
	"clippy": {},
}

func validateAndPrepareNpmHostReadOnlyArgs(args []string) (hostReadOnlyCommandAnalysis, error) {
	if len(args) == 0 {
		return nil, newHostReadOnlyBlockedError("blocked command: npm subcommand is required")
	}
	if args[0] == "test" {
		return hostReadOnlyNoopAnalysis{}, nil
	}
	if args[0] == "run" && len(args) >= 2 {
		if _, ok := allowedNpmRunHostReadOnlyScripts[args[1]]; ok {
			return hostReadOnlyNoopAnalysis{}, nil
		}
	}
	return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: npm %s is not allowed in host_readonly", strings.Join(args, " ")))
}

func validateAndPrepareCargoHostReadOnlyArgs(args []string) (hostReadOnlyCommandAnalysis, error) {
	if len(args) == 0 {
		return nil, newHostReadOnlyBlockedError("blocked command: cargo subcommand is required")
	}
	if _, ok := allowedCargoHostReadOnlySubcommands[args[0]]; ok {
		return hostReadOnlyNoopAnalysis{}, nil
	}
	return nil, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: cargo %s is not allowed in host_readonly", args[0]))
}
