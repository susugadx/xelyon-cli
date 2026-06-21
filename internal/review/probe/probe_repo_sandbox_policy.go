package probe

import "strings"

type repoSandboxCommandSpec struct {
	validateAndCollectPathArgs func(args []string) ([]string, error)
}

var repoSandboxCommandSpecs = map[string]repoSandboxCommandSpec{
	"python3": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return validateAndCollectRepoSandboxPythonPathArgs("python3", args)
		},
	},
	"python": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return validateAndCollectRepoSandboxPythonPathArgs("python", args)
		},
	},
	"go": {
		validateAndCollectPathArgs: validateAndCollectRepoSandboxGoPathArgs,
	},
	"cat": {
		validateAndCollectPathArgs: validateAndCollectRepoSandboxCatPathArgs,
	},
	"ls": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return collectRepoSandboxLSPathArgs(args), nil
		},
	},
	"find": {
		validateAndCollectPathArgs: validateAndCollectRepoSandboxFindPathArgs,
	},
	"rg": {
		validateAndCollectPathArgs: validateAndCollectRepoSandboxRGPathArgs,
	},
	"grep": {
		validateAndCollectPathArgs: func(args []string) ([]string, error) {
			return collectSearchCommandPathCandidates("grep", args), nil
		},
	},
}

func analyzeRepoSandboxCommand(command string, args []string) ([]string, error) {
	if strings.ContainsAny(command, `/\`) {
		return nil, newBlockedCommandErrorf("command path is not allowed in repo_sandbox: %s", command)
	}

	spec, ok := repoSandboxCommandSpecs[command]
	if !ok {
		return nil, newBlockedCommandErrorf("%s is not allowed in repo_sandbox", command)
	}

	return spec.validateAndCollectPathArgs(args)
}
