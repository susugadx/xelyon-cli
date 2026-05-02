package review

import (
	"fmt"
	"strings"
)

func validateHostReadOnlyCommandPolicy(command string, args []string) error {
	if strings.ContainsAny(command, `/\`) {
		return fmt.Errorf("blocked command: command path is not allowed in host_readonly: %s", command)
	}

	switch command {
	case "git":
		return validateGitHostReadOnlyArgs(args)
	case "rg", "grep", "ls", "cat":
		return nil
	case "find":
		return validateFindHostReadOnlyArgs(args)
	case "sed":
		if len(args) == 0 || args[0] != "-n" {
			return fmt.Errorf("blocked command: sed only supports '-n' in host_readonly")
		}
		return nil
	case "go":
		return validateGoHostReadOnlyArgs(args)
	case "npm":
		return validateNpmHostReadOnlyArgs(args)
	case "cargo":
		return validateCargoHostReadOnlyArgs(args)
	default:
		return fmt.Errorf("blocked command: %s is not allowed in host_readonly", command)
	}
}

func isBlockedFlagArg(arg string, blocked []string) bool {
	for _, flag := range blocked {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}
