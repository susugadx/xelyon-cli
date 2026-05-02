package review

import (
	"fmt"
	"strings"
)

type parsedGitHostReadOnlyArgs struct {
	subcommand        string
	postSubcommandArg []string
}

var (
	allowedGitHostReadOnlySubcommands = map[string]struct{}{
		"status": {},
		"diff":   {},
		"show":   {},
		"grep":   {},
	}
	blockedGitHostReadOnlyPostSubcommandFlags = []string{
		"--ext-diff",
		"--external-diff",
		"--output",
	}
	allowedGitHostReadOnlyGlobalOptions = map[string]struct{}{
		"--no-optional-locks":    {},
		"--no-replace-objects":   {},
		"--literal-pathspecs":    {},
		"--no-literal-pathspecs": {},
		"--glob-pathspecs":       {},
		"--noglob-pathspecs":     {},
		"--icase-pathspecs":      {},
	}
)

func validateAndPrepareGitHostReadOnlyArgs(args []string) (hostReadOnlyCommandState, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandState{}, newHostReadOnlyBlockedError("blocked command: git subcommand is required")
	}

	parsed, err := parseGitHostReadOnlyArgs(args)
	if err != nil {
		return hostReadOnlyCommandState{}, err
	}

	if err := validateGitHostReadOnlySubcommand(parsed.subcommand); err != nil {
		return hostReadOnlyCommandState{}, err
	}

	if err := validateGitHostReadOnlyPostSubcommandArgs(parsed.postSubcommandArg); err != nil {
		return hostReadOnlyCommandState{}, err
	}

	return hostReadOnlyCommandState{
		gitParsed: parsed,
	}, nil
}

func parseGitHostReadOnlyArgs(args []string) (parsedGitHostReadOnlyArgs, error) {
	subcommandIndex, err := findGitHostReadOnlySubcommandIndex(args)
	if err != nil {
		return parsedGitHostReadOnlyArgs{}, err
	}
	return parsedGitHostReadOnlyArgs{
		subcommand:        args[subcommandIndex],
		postSubcommandArg: args[subcommandIndex+1:],
	}, nil
}

func findGitHostReadOnlySubcommandIndex(args []string) (int, error) {
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			return i, nil
		}
		if arg == "--" {
			return -1, newHostReadOnlyBlockedError("blocked command: git global separator -- is not allowed before subcommand in host_readonly")
		}
		if err := validateGitHostReadOnlyGlobalOption(arg); err != nil {
			return -1, err
		}
	}
	return -1, newHostReadOnlyBlockedError("blocked command: git subcommand is required")
}

func validateGitHostReadOnlySubcommand(subcommand string) error {
	if _, ok := allowedGitHostReadOnlySubcommands[subcommand]; !ok {
		return newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: git %s is not allowed in host_readonly", subcommand))
	}
	return nil
}

func validateGitHostReadOnlyPostSubcommandArgs(args []string) error {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedGitHostReadOnlyPostSubcommandFlags) {
			return newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: git argument %s is not allowed in host_readonly", arg))
		}
	}
	return nil
}

func validateGitHostReadOnlyGlobalOption(arg string) error {
	switch {
	case isGitConfigOverride(arg):
		return newHostReadOnlyBlockedError("blocked command: git config override is not allowed in host_readonly")
	case arg == "-p" || arg == "-P" || arg == "--paginate" || arg == "--no-pager":
		return newHostReadOnlyBlockedError("blocked command: git pager option is not allowed in host_readonly")
	case arg == "--pager" || strings.HasPrefix(arg, "--pager="):
		return newHostReadOnlyBlockedError("blocked command: git pager option is not allowed in host_readonly")
	case isGitPathOverride(arg):
		return newHostReadOnlyBlockedError("blocked command: git path override is not allowed in host_readonly")
	case isAllowedGitHostReadOnlyGlobalOption(arg):
		return nil
	default:
		return newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: git global option %s is not allowed in host_readonly", arg))
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

func isAllowedGitHostReadOnlyGlobalOption(arg string) bool {
	_, ok := allowedGitHostReadOnlyGlobalOptions[arg]
	return ok
}

func extractGitHostReadOnlyPathArgs(_ []string, state hostReadOnlyCommandState) ([]string, error) {
	return extractArgsAfterDoubleDash(state.gitParsed.postSubcommandArg), nil
}
