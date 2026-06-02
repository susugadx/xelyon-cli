package probe

import (
	"strings"
)

type hostReadOnlyCommandPolicyResult struct {
	pathArgs []string
}

func newHostReadOnlyPolicyResult(pathArgs []string) hostReadOnlyCommandPolicyResult {
	return hostReadOnlyCommandPolicyResult{
		pathArgs: append([]string(nil), pathArgs...),
	}
}

func newHostReadOnlyNoPathPolicyResult() hostReadOnlyCommandPolicyResult {
	return hostReadOnlyCommandPolicyResult{}
}

type hostReadOnlyCommandSpec struct {
	validateAndPrepare func(args []string) (hostReadOnlyCommandPolicyResult, error)
}

var hostReadOnlyCommandSpecs = map[string]hostReadOnlyCommandSpec{
	"git": {
		validateAndPrepare: validateAndPrepareGitHostReadOnlyArgs,
	},
	"rg": {
		validateAndPrepare: validateAndPrepareRGHostReadOnlyArgs,
	},
	"grep": {
		validateAndPrepare: validateAndPrepareGrepHostReadOnlyArgs,
	},
	"ls": {
		validateAndPrepare: validateAndPrepareLSHostReadOnlyArgs,
	},
	"cat": {
		validateAndPrepare: validateAndPrepareCatHostReadOnlyArgs,
	},
	"find": {
		validateAndPrepare: validateAndPrepareFindHostReadOnlyArgs,
	},
	"sed": {
		validateAndPrepare: validateAndPrepareSEDHostReadOnlyArgs,
	},
	"go": {
		validateAndPrepare: validateAndPrepareGoHostReadOnlyArgs,
	},
	"npm": {
		validateAndPrepare: validateAndPrepareNpmHostReadOnlyArgs,
	},
	"cargo": {
		validateAndPrepare: validateAndPrepareCargoHostReadOnlyArgs,
	},
}

type analyzedHostReadOnlyCommand struct {
	pathArgs []string
}

func analyzeHostReadOnlyCommand(command string, args []string) (analyzedHostReadOnlyCommand, error) {
	if strings.ContainsAny(command, `/\`) {
		return analyzedHostReadOnlyCommand{}, newBlockedCommandErrorf("command path is not allowed in host_readonly: %s", command)
	}

	spec, ok := hostReadOnlyCommandSpecs[command]
	if !ok {
		return analyzedHostReadOnlyCommand{}, newBlockedCommandErrorf("%s is not allowed in host_readonly", command)
	}

	policyResult, err := spec.validateAndPrepare(args)
	if err != nil {
		return analyzedHostReadOnlyCommand{}, err
	}

	return analyzedHostReadOnlyCommand(policyResult), nil
}

func validateHostReadOnlyCommandPolicy(command string, args []string) error {
	_, err := analyzeHostReadOnlyCommand(command, args)
	return err
}

func validateAndPrepareSEDHostReadOnlyArgs(args []string) (hostReadOnlyCommandPolicyResult, error) {
	if len(args) == 0 || args[0] != "-n" {
		return hostReadOnlyCommandPolicyResult{}, newBlockedCommandErrorf("sed only supports '-n' in host_readonly")
	}
	pathArgs, err := collectSEDHostReadOnlyPathArgs(args[1:])
	if err != nil {
		return hostReadOnlyCommandPolicyResult{}, err
	}
	return newHostReadOnlyPolicyResult(pathArgs), nil
}

func collectSEDHostReadOnlyPathArgs(args []string) ([]string, error) {
	pathArgs := make([]string, 0, len(args))
	scriptSeen := false
	optionsTerminated := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		if !optionsTerminated && arg == "--" {
			optionsTerminated = true
			continue
		}

		if !optionsTerminated && strings.HasPrefix(arg, "-") && arg != "-" {
			result, err := parseSEDHostReadOnlyOption(args, &i)
			if err != nil {
				return nil, err
			}
			if result.script {
				if err := validateSEDReadOnlyPrintScript(result.scriptText); err != nil {
					return nil, err
				}
				scriptSeen = true
			}
			pathArgs = append(pathArgs, result.pathArgs...)
			continue
		}

		if !scriptSeen {
			if err := validateSEDReadOnlyPrintScript(arg); err != nil {
				return nil, err
			}
			scriptSeen = true
			continue
		}
		if arg == "-" {
			return nil, newBlockedCommandArgError("sed", arg)
		}
		pathArgs = append(pathArgs, arg)
	}

	if !scriptSeen {
		return nil, newBlockedCommandErrorf("sed script is required in host_readonly")
	}
	return pathArgs, nil
}

type parsedSEDHostReadOnlyOption struct {
	script     bool
	scriptText string
	pathArgs   []string
}

func parseSEDHostReadOnlyOption(args []string, index *int) (parsedSEDHostReadOnlyOption, error) {
	arg := args[*index]
	switch {
	case arg == "-n":
		return parsedSEDHostReadOnlyOption{}, nil
	case arg == "-e" || arg == "--expression":
		if *index+1 >= len(args) {
			return parsedSEDHostReadOnlyOption{}, newBlockedCommandErrorf("sed option %s requires a script argument in host_readonly", arg)
		}
		*index = *index + 1
		return parsedSEDHostReadOnlyOption{script: true, scriptText: args[*index]}, nil
	case strings.HasPrefix(arg, "-e") && arg != "-e":
		return parsedSEDHostReadOnlyOption{script: true, scriptText: strings.TrimPrefix(arg, "-e")}, nil
	case strings.HasPrefix(arg, "--expression="):
		return parsedSEDHostReadOnlyOption{script: true, scriptText: strings.TrimPrefix(arg, "--expression=")}, nil
	case arg == "-f" || arg == "--file":
		if *index+1 >= len(args) {
			return parsedSEDHostReadOnlyOption{}, newBlockedCommandErrorf("sed option %s requires a script file path in host_readonly", arg)
		}
		return parsedSEDHostReadOnlyOption{}, newBlockedCommandArgError("sed", arg)
	case strings.HasPrefix(arg, "-f") && arg != "-f":
		return parsedSEDHostReadOnlyOption{}, newBlockedCommandArgError("sed", arg)
	case strings.HasPrefix(arg, "--file="):
		return parsedSEDHostReadOnlyOption{}, newBlockedCommandArgError("sed", arg)
	default:
		return parsedSEDHostReadOnlyOption{}, newBlockedCommandOptionError("sed", arg)
	}
}

func validateSEDReadOnlyPrintScript(script string) error {
	if strings.TrimSpace(script) == "" {
		return newBlockedCommandErrorf("sed script is required in host_readonly")
	}

	commandsSeen := 0
	for _, command := range strings.FieldsFunc(script, isSEDCommandSeparator) {
		if strings.TrimSpace(command) != "" {
			commandsSeen++
		}
		if err := validateSEDReadOnlyPrintCommand(command); err != nil {
			return err
		}
	}
	if commandsSeen == 0 {
		return newBlockedCommandErrorf("sed script is required in host_readonly")
	}
	return nil
}

func isSEDCommandSeparator(r rune) bool {
	return r == ';' || r == '\n'
}

func validateSEDReadOnlyPrintCommand(command string) error {
	trimmed := strings.TrimSpace(command)
	if trimmed == "" {
		return nil
	}
	if strings.ContainsAny(trimmed, "{}") {
		return newBlockedCommandErrorf("sed script %q is not allowed in host_readonly", command)
	}
	if !strings.HasSuffix(trimmed, "p") {
		return newBlockedCommandErrorf("sed script %q is not a read-only print command in host_readonly", command)
	}

	address := strings.TrimSpace(strings.TrimSuffix(trimmed, "p"))
	if address == "" {
		return nil
	}
	for _, r := range address {
		if !isAllowedSEDPrintAddressRune(r) {
			return newBlockedCommandErrorf("sed script %q is not a read-only print command in host_readonly", command)
		}
	}
	return nil
}

func isAllowedSEDPrintAddressRune(r rune) bool {
	return (r >= '0' && r <= '9') ||
		r == '$' ||
		r == ',' ||
		r == '+' ||
		r == '-' ||
		r == '~' ||
		r == ' ' ||
		r == '\t'
}

func isBlockedFlagArg(arg string, blocked []string) bool {
	for _, flag := range blocked {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}
