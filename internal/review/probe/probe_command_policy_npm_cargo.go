package probe

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

var blockedNpmHostReadOnlyFlags = []string{
	"--prefix",
	"--workspace",
	"--workspaces",
	"--include-workspace-root",
	"--userconfig",
	"--globalconfig",
	"--script-shell",
	"--cache",
	"--tmp",
	"--nodedir",
	"--node-gyp",
	"--global",
	"--location",
	"-C",
	"-w",
	"-g",
}

var blockedCargoHostReadOnlyFlags = []string{
	"--manifest-path",
	"--target-dir",
	"--config",
	"--registry",
	"--index",
}

func validateAndPrepareNpmHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("npm subcommand is required")
	}
	if err := validateNpmHostReadOnlyArgsBeforeSeparator(args); err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
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
		if err := validateCargoHostReadOnlyArgsBeforeSeparator(args[1:]); err != nil {
			return hostReadOnlyCommandPolicyResult{}, err
		}
		return newHostReadOnlyNoPathPolicyResult(), nil
	}
	return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("cargo %s is not allowed in host_readonly", args[0])
}

func validateNpmHostReadOnlyArgsBeforeSeparator(args []string) error {
	for _, arg := range argsBeforeDoubleDash(args) {
		if isBlockedNpmHostReadOnlyArg(arg) {
			return newBlockedCommandArgError("npm", arg)
		}
	}
	return nil
}

func isBlockedNpmHostReadOnlyArg(arg string) bool {
	return isBlockedFlagArg(arg, blockedNpmHostReadOnlyFlags) ||
		strings.HasPrefix(arg, "-C") ||
		strings.HasPrefix(arg, "-w") ||
		arg == "-g"
}

func validateCargoHostReadOnlyArgsBeforeSeparator(args []string) error {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			return nil
		}
		if isBlockedFlagArg(arg, blockedCargoHostReadOnlyFlags) {
			return newBlockedCommandArgError("cargo", arg)
		}
		if isCargoTargetFlag(arg) {
			targetValue := cargoFlagAttachedValue(arg, "--target")
			if targetValue == "" && i+1 < len(args) {
				targetValue = args[i+1]
			}
			if isPathLikeGoOrCargoValue(targetValue) {
				return newBlockedCommandArgError("cargo", arg)
			}
		}
	}
	return nil
}

func argsBeforeDoubleDash(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			return args[:i]
		}
	}
	return args
}

func isCargoTargetFlag(arg string) bool {
	return arg == "--target" || strings.HasPrefix(arg, "--target=")
}

func cargoFlagAttachedValue(arg, flag string) string {
	if strings.HasPrefix(arg, flag+"=") {
		return strings.TrimPrefix(arg, flag+"=")
	}
	return ""
}

func isPathLikeGoOrCargoValue(value string) bool {
	return strings.ContainsAny(value, `/\`)
}
