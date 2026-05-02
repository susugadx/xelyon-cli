package review

import (
	"fmt"
	"strings"
)

func validateNpmHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: npm subcommand is required")
	}
	if args[0] == "test" {
		return nil
	}
	if args[0] == "run" && len(args) >= 2 {
		switch args[1] {
		case "test", "lint":
			return nil
		}
	}
	return fmt.Errorf("blocked command: npm %s is not allowed in host_readonly", strings.Join(args, " "))
}

func validateCargoHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: cargo subcommand is required")
	}
	switch args[0] {
	case "test", "clippy":
		return nil
	default:
		return fmt.Errorf("blocked command: cargo %s is not allowed in host_readonly", args[0])
	}
}
