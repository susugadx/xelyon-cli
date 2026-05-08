package review

import (
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
		"--no-index",
		"--output",
		"--open-files-in-pager",
		"--textconv",
		"--recurse-submodules",
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
	gitGrepShortOptionsWithValue = byteSet(
		'A', 'B', 'C', 'e', 'f', 'm', 'O',
	)
)

func validateAndPrepareGitHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("git subcommand is required")
	}

	parsed, err := parseGitHostReadOnlyArgs(args)
	if err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
	}

	if err := validateGitHostReadOnlySubcommand(parsed.subcommand); err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
	}

	if err := validateGitHostReadOnlyPostSubcommandArgs(parsed.subcommand, parsed.postSubcommandArg); err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
	}

	return newHostReadOnlyPolicyResult(extractArgsAfterDoubleDash(parsed.postSubcommandArg)), nil
}

func extractArgsAfterDoubleDash(args []string) []string {
	for i, arg := range args {
		if arg == "--" {
			if i+1 >= len(args) {
				return nil
			}
			return append([]string(nil), args[i+1:]...)
		}
	}
	return nil
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
			return -1, newBlockedCommandErrorf("git global separator -- is not allowed before subcommand in host_readonly")
		}
		if err := validateGitHostReadOnlyGlobalOption(arg); err != nil {
			return -1, err
		}
	}
	return -1, newBlockedCommandErrorf("git subcommand is required")
}

func validateGitHostReadOnlySubcommand(subcommand string) error {
	if _, ok := allowedGitHostReadOnlySubcommands[subcommand]; !ok {
		return newBlockedCommandErrorf("git %s is not allowed in host_readonly", subcommand)
	}
	return nil
}

func validateGitHostReadOnlyPostSubcommandArgs(subcommand string, args []string) error {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedGitHostReadOnlyPostSubcommandFlags) {
			return newBlockedCommandArgError("git", arg)
		}
		if isBlockedGitPagerOpenArg(arg) {
			return newBlockedCommandArgError("git", arg)
		}
		if subcommand == "grep" && isBlockedGitGrepPatternFileArg(arg) {
			return newBlockedCommandArgError("git", arg)
		}
	}
	return nil
}

func isBlockedGitPagerOpenArg(arg string) bool {
	if arg == "-O" || strings.HasPrefix(arg, "-O") {
		return true
	}
	tokens, ok := parseProbeShortOptions(arg, gitGrepShortOptionsWithValue)
	return ok && probeShortOptionsContain(tokens, 'O')
}

func isBlockedGitGrepPatternFileArg(arg string) bool {
	if arg == "-f" ||
		strings.HasPrefix(arg, "-f") ||
		arg == "--file" ||
		strings.HasPrefix(arg, "--file=") {
		return true
	}
	tokens, ok := parseProbeShortOptions(arg, gitGrepShortOptionsWithValue)
	return ok && probeShortOptionsContain(tokens, 'f')
}

func validateGitHostReadOnlyGlobalOption(arg string) error {
	switch {
	case isGitConfigOverride(arg):
		return newBlockedCommandErrorf("git config override is not allowed in host_readonly")
	case arg == "-p" || arg == "-P" || arg == "--paginate" || arg == "--no-pager":
		return newBlockedCommandErrorf("git pager option is not allowed in host_readonly")
	case arg == "--pager" || strings.HasPrefix(arg, "--pager="):
		return newBlockedCommandErrorf("git pager option is not allowed in host_readonly")
	case isGitPathOverride(arg):
		return newBlockedCommandErrorf("git path override is not allowed in host_readonly")
	case isAllowedGitHostReadOnlyGlobalOption(arg):
		return nil
	default:
		return newBlockedCommandErrorf("git global option %s is not allowed in host_readonly", arg)
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
