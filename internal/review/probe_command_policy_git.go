package review

import (
	"fmt"
	"strings"
)

func validateGitHostReadOnlyArgs(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("blocked command: git subcommand is required")
	}

	subcommandIndex, err := findGitSubcommandIndex(args)
	if err != nil {
		return err
	}

	for _, arg := range args[subcommandIndex+1:] {
		if isBlockedFlagArg(arg, []string{"--ext-diff", "--external-diff", "--output"}) {
			return fmt.Errorf("blocked command: git argument %s is not allowed in host_readonly", arg)
		}
	}

	switch args[subcommandIndex] {
	case "status", "diff", "show", "grep":
		return nil
	default:
		return fmt.Errorf("blocked command: git %s is not allowed in host_readonly", args[subcommandIndex])
	}
}

func findGitSubcommandIndex(args []string) (int, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return i, nil
		}
		if arg == "--" {
			if i+1 >= len(args) {
				return -1, fmt.Errorf("blocked command: git subcommand is required")
			}
			return i + 1, nil
		}
		if err := validateGitGlobalOption(arg); err != nil {
			return -1, err
		}
	}
	return -1, fmt.Errorf("blocked command: git subcommand is required")
}

func validateGitGlobalOption(arg string) error {
	switch {
	case isGitConfigOverride(arg):
		return fmt.Errorf("blocked command: git config override is not allowed in host_readonly")
	case arg == "-p" || arg == "-P" || arg == "--paginate" || arg == "--no-pager":
		return fmt.Errorf("blocked command: git pager option is not allowed in host_readonly")
	case arg == "--pager" || strings.HasPrefix(arg, "--pager="):
		return fmt.Errorf("blocked command: git pager option is not allowed in host_readonly")
	case isGitPathOverride(arg):
		return fmt.Errorf("blocked command: git path override is not allowed in host_readonly")
	case isAllowedGitGlobalOption(arg):
		return nil
	default:
		return fmt.Errorf("blocked command: git global option %s is not allowed in host_readonly", arg)
	}
}

func isGitConfigOverride(arg string) bool {
	switch {
	case arg == "-c":
		return true
	case len(arg) > 2 && strings.HasPrefix(arg, "-c") && arg[1] == 'c':
		return true
	case strings.HasPrefix(arg, "--config="), arg == "--config", strings.HasPrefix(arg, "--config-env="), arg == "--config-env":
		return true
	default:
		return false
	}
}

func isGitPathOverride(arg string) bool {
	return arg == "-C" ||
		strings.HasPrefix(arg, "-C") ||
		arg == "--git-dir" ||
		strings.HasPrefix(arg, "--git-dir=") ||
		arg == "--work-tree" ||
		strings.HasPrefix(arg, "--work-tree=") ||
		arg == "--namespace" ||
		strings.HasPrefix(arg, "--namespace=") ||
		arg == "--super-prefix" ||
		strings.HasPrefix(arg, "--super-prefix=") ||
		arg == "--exec-path" ||
		strings.HasPrefix(arg, "--exec-path=")
}

func isAllowedGitGlobalOption(arg string) bool {
	switch arg {
	case "--no-optional-locks", "--no-replace-objects", "--literal-pathspecs", "--no-literal-pathspecs", "--glob-pathspecs", "--noglob-pathspecs", "--icase-pathspecs":
		return true
	default:
		return false
	}
}
