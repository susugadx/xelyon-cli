package review

import (
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

func validateAndPrepareNpmHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("npm subcommand is required")
	}
	if args[0] == "test" {
		return newHostReadOnlyNoPathPolicyResult(), nil
	}
	if args[0] == "run" && len(args) >= 2 {
		if _, ok := allowedNpmRunHostReadOnlyScripts[args[1]]; ok {
			return newHostReadOnlyNoPathPolicyResult(), nil
		}
	}
	return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("npm %s is not allowed in host_readonly", strings.Join(args, " "))
}

func validateAndPrepareCargoHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("cargo subcommand is required")
	}
	if _, ok := allowedCargoHostReadOnlySubcommands[args[0]]; ok {
		return newHostReadOnlyNoPathPolicyResult(), nil
	}
	return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("cargo %s is not allowed in host_readonly", args[0])
}
