package review

import (
	"fmt"
	"strings"
)

type hostReadOnlyCommandValidator func(args []string) error

var hostReadOnlyCommandValidators = map[string]hostReadOnlyCommandValidator{
	"git":   validateGitHostReadOnlyArgs,
	"rg":    validateRGHostReadOnlyArgs,
	"grep":  validateGrepHostReadOnlyArgs,
	"ls":    validateLSHostReadOnlyArgs,
	"cat":   validateCatHostReadOnlyArgs,
	"find":  validateFindHostReadOnlyArgs,
	"sed":   validateSEDHostReadOnlyArgs,
	"go":    validateGoHostReadOnlyArgs,
	"npm":   validateNpmHostReadOnlyArgs,
	"cargo": validateCargoHostReadOnlyArgs,
}

func validateHostReadOnlyCommandPolicy(command string, args []string) error {
	if strings.ContainsAny(command, `/\`) {
		return fmt.Errorf("blocked command: command path is not allowed in host_readonly: %s", command)
	}

	validator, ok := hostReadOnlyCommandValidators[command]
	if !ok {
		return fmt.Errorf("blocked command: %s is not allowed in host_readonly", command)
	}
	return validator(args)
}

func validateSEDHostReadOnlyArgs(args []string) error {
	if len(args) == 0 || args[0] != "-n" {
		return fmt.Errorf("blocked command: sed only supports '-n' in host_readonly")
	}
	return nil
}

func isBlockedFlagArg(arg string, blocked []string) bool {
	for _, flag := range blocked {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}
