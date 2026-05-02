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

func validateNpmHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: npm subcommand is required")
	}
	if args[0] == "test" {
		return nil
	}
	if args[0] == "run" && len(args) >= 2 {
		if _, ok := allowedNpmRunHostReadOnlyScripts[args[1]]; ok {
			return nil
		}
	}
	return fmt.Errorf("blocked command: npm %s is not allowed in host_readonly", strings.Join(args, " "))
}

func validateCargoHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: cargo subcommand is required")
	}
	if _, ok := allowedCargoHostReadOnlySubcommands[args[0]]; ok {
		return nil
	}
	return fmt.Errorf("blocked command: cargo %s is not allowed in host_readonly", args[0])
}
