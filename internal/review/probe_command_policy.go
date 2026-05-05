package review

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
	return newHostReadOnlyNoPathPolicyResult(), nil
}

func isBlockedFlagArg(arg string, blocked []string) bool {
	for _, flag := range blocked {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}
