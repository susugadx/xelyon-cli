package review

import "strings"

type scratchOnlyCommandSpec struct {
	validateAndCollectPathArgs func(args []string) ([]string, error)
}

var scratchOnlyCommandSpecs = map[string]scratchOnlyCommandSpec{
	"python3": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return validateAndCollectScratchPythonPathArgs("python3", args)
		},
	},
	"python": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return validateAndCollectScratchPythonPathArgs("python", args)
		},
	},
	"go": {
		validateAndCollectPathArgs: validateAndCollectScratchGoPathArgs,
	},
	"cat": {
		validateAndCollectPathArgs: validateAndCollectScratchCatPathArgs,
	},
	"ls": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return collectScratchLSPathArgs(args), nil
		},
	},
	"rg": {
		validateAndCollectPathArgs: validateAndCollectScratchRGPathArgs,
	},
	"grep": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return collectSearchCommandPathCandidates("grep", args), nil
		},
	},
}

func analyzeScratchOnlyCommand(command string, args []string) ([]string, error) {
	if strings.ContainsAny(command, `/\\`) {
		return nil, newBlockedCommandErrorf("command path is not allowed in scratch_only: %s", command)
	}

	spec, ok := scratchOnlyCommandSpecs[command]
	if !ok {
		return nil, newBlockedCommandErrorf("%s is not allowed in scratch_only", command)
	}

	return spec.validateAndCollectPathArgs(args)
}

func validateAndCollectScratchPythonPathArgs(command string, args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, newBlockedCommandErrorf("%s requires a script path in scratch_only", command)
	}
	if strings.HasPrefix(args[0], "-") {
		return nil, newBlockedCommandArgError(command, args[0])
	}
	if len(args) > 1 {
		return nil, newBlockedCommandErrorf("%s accepts only one script path in scratch_only", command)
	}
	return []string{args[0]}, nil
}

func validateAndCollectScratchGoPathArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, newBlockedCommandErrorf("go subcommand is required")
	}
	if args[0] != "run" {
		return nil, newBlockedCommandErrorf("go %s is not allowed in scratch_only", args[0])
	}
	if len(args) != 2 {
		return nil, newBlockedCommandErrorf("go run requires exactly one .go file path in scratch_only")
	}
	if strings.HasPrefix(args[1], "-") {
		return nil, newBlockedCommandArgError("go", args[1])
	}
	if !strings.HasSuffix(args[1], ".go") {
		return nil, newBlockedCommandErrorf("go run target %q must be a .go file in scratch_only", args[1])
	}
	return []string{args[1]}, nil
}

func validateAndCollectScratchCatPathArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, newBlockedCommandErrorf("cat requires at least one path argument in scratch_only")
	}
	for _, arg := range args {
		if arg == catStdinArg {
			return nil, newBlockedCommandArgError("cat", arg)
		}
		if strings.HasPrefix(arg, "-") {
			return nil, newBlockedCommandOptionError("cat", arg)
		}
	}
	return append([]string(nil), args...), nil
}

func collectScratchLSPathArgs(args []string) []string {
	if len(args) == 0 {
		return nil
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		paths = append(paths, arg)
	}
	return paths
}

func validateAndCollectScratchRGPathArgs(args []string) ([]string, error) {
	for _, arg := range args {
		if isBlockedRGHostReadOnlyArg(arg) {
			return nil, newBlockedCommandArgError("rg", arg)
		}
	}
	return collectSearchCommandPathCandidates("rg", args), nil
}
