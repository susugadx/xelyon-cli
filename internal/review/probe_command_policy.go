package review

import (
	"fmt"
	"strings"
)

type hostReadOnlyCommandAnalysis interface {
	hostReadOnlyCommandAnalysis()
}

type hostReadOnlyNoopAnalysis struct{}

func (hostReadOnlyNoopAnalysis) hostReadOnlyCommandAnalysis() {}

type gitHostReadOnlyAnalysis struct {
	parsed parsedGitHostReadOnlyArgs
}

func (gitHostReadOnlyAnalysis) hostReadOnlyCommandAnalysis() {}

type findHostReadOnlyAnalysis struct {
	parsed parsedFindHostReadOnlyArgs
}

func (findHostReadOnlyAnalysis) hostReadOnlyCommandAnalysis() {}

type hostReadOnlyCommandSpec struct {
	validateAndPrepare func(args []string) (hostReadOnlyCommandAnalysis, error)
	extractPathArgs    func(args []string, analysis hostReadOnlyCommandAnalysis) ([]string, error)
}

var hostReadOnlyCommandSpecs = map[string]hostReadOnlyCommandSpec{
	"git": {
		validateAndPrepare: validateAndPrepareGitHostReadOnlyArgs,
		extractPathArgs:    extractGitHostReadOnlyPathArgs,
	},
	"rg": {
		validateAndPrepare: validateAndPrepareRGHostReadOnlyArgs,
		extractPathArgs:    extractArgsAfterDoubleDashFromCommandArgs,
	},
	"grep": {
		validateAndPrepare: validateAndPrepareGrepHostReadOnlyArgs,
		extractPathArgs:    extractArgsAfterDoubleDashFromCommandArgs,
	},
	"ls": {
		validateAndPrepare: validateAndPrepareLSHostReadOnlyArgs,
		extractPathArgs:    extractLSPathArgsFromCommandArgs,
	},
	"cat": {
		validateAndPrepare: validateAndPrepareCatHostReadOnlyArgs,
		extractPathArgs:    extractCatPathArgsFromCommandArgs,
	},
	"find": {
		validateAndPrepare: validateAndPrepareFindHostReadOnlyArgs,
		extractPathArgs:    extractFindPathRootsFromCommandArgs,
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
		return analyzedHostReadOnlyCommand{}, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: command path is not allowed in host_readonly: %s", command))
	}

	spec, ok := hostReadOnlyCommandSpecs[command]
	if !ok {
		return analyzedHostReadOnlyCommand{}, newHostReadOnlyBlockedError(fmt.Sprintf("blocked command: %s is not allowed in host_readonly", command))
	}

	analysis, err := spec.validateAndPrepare(args)
	if err != nil {
		return analyzedHostReadOnlyCommand{}, err
	}

	pathArgs := []string(nil)
	if spec.extractPathArgs != nil {
		pathArgs, err = spec.extractPathArgs(args, analysis)
		if err != nil {
			return analyzedHostReadOnlyCommand{}, err
		}
	}

	return analyzedHostReadOnlyCommand{
		pathArgs: pathArgs,
	}, nil
}

func validateHostReadOnlyCommandPolicy(command string, args []string) error {
	_, err := analyzeHostReadOnlyCommand(command, args)
	return err
}

func validateAndPrepareSEDHostReadOnlyArgs(args []string) (hostReadOnlyCommandAnalysis, error) {
	if len(args) == 0 || args[0] != "-n" {
		return nil, newHostReadOnlyBlockedError("blocked command: sed only supports '-n' in host_readonly")
	}
	return hostReadOnlyNoopAnalysis{}, nil
}

func isBlockedFlagArg(arg string, blocked []string) bool {
	for _, flag := range blocked {
		if arg == flag || strings.HasPrefix(arg, flag+"=") {
			return true
		}
	}
	return false
}
