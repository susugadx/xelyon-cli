package probe

import (
	"path/filepath"
	"strings"
)

func validateAndCollectRepoSandboxGoPathArgs(args []string) ([]string, error) {
	if len(args) == 0 {
		return nil, newBlockedCommandErrorf("go subcommand is required")
	}
	if args[0] == "env" && containsGoEnvWriteFlag(args[1:]) {
		return nil, newBlockedCommandErrorf("go env -w is not allowed in repo_sandbox")
	}
	if !isAllowedRepoSandboxGoSubcommand(args[0]) {
		return nil, newBlockedCommandErrorf("go %s is not allowed in repo_sandbox", args[0])
	}
	if err := validateRepoSandboxBlockedGoWrapperFlags(args[1:]); err != nil {
		return nil, err
	}

	switch args[0] {
	case "run":
		return collectRepoSandboxGoRunPathArgs(args[1:]), nil
	case "test", "build", "vet":
		return collectRepoSandboxGoLocalPathArgs(args[1:]), nil
	default:
		return nil, nil
	}
}

func containsGoEnvWriteFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-w" || strings.HasPrefix(arg, "-w=") {
			return true
		}
	}
	return false
}

func isAllowedRepoSandboxGoSubcommand(subcommand string) bool {
	switch subcommand {
	case "test", "build", "vet", "run":
		return true
	default:
		return false
	}
}

var blockedRepoSandboxGoWrapperFlags = []string{
	"-exec",
	"--exec",
	"-toolexec",
	"--toolexec",
	"-vettool",
	"--vettool",
}

func validateRepoSandboxBlockedGoWrapperFlags(args []string) error {
	for _, arg := range args {
		if isBlockedFlagArg(arg, blockedRepoSandboxGoWrapperFlags) {
			return newBlockedCommandErrorf("go argument %s is not allowed in repo_sandbox", arg)
		}
	}
	return nil
}

func collectRepoSandboxGoRunPathArgs(args []string) []string {
	paths := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			continue
		}

		if pathValue, consumed := collectRepoSandboxGoPathFlagValue(arg, args, &i); consumed {
			if pathValue != "" {
				paths = append(paths, pathValue)
			}
			continue
		}
		if strings.HasPrefix(arg, "-") {
			consumeRepoSandboxGoNonPathFlagValue(arg, args, &i)
			continue
		}

		if strings.HasSuffix(arg, ".go") {
			for ; i < len(args); i++ {
				if strings.HasSuffix(args[i], ".go") {
					paths = append(paths, args[i])
					continue
				}
				break
			}
			return paths
		}
		if isLocalGoPackagePathArg(arg) {
			paths = append(paths, arg)
		}
		return paths
	}
	return paths
}

func collectRepoSandboxGoLocalPathArgs(args []string) []string {
	paths := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			continue
		}

		if pathValue, consumed := collectRepoSandboxGoPathFlagValue(arg, args, &i); consumed {
			if pathValue != "" {
				paths = append(paths, pathValue)
			}
			continue
		}
		if arg == "-args" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			consumeRepoSandboxGoNonPathFlagValue(arg, args, &i)
			continue
		}
		if isLocalGoPackagePathArg(arg) {
			paths = append(paths, arg)
		}
	}
	return paths
}

func collectRepoSandboxGoPathFlagValue(arg string, args []string, index *int) (string, bool) {
	name, value, hasValue := splitRepoSandboxGoFlagValue(arg)
	if !isRepoSandboxGoPathFlag(name) {
		return "", false
	}
	if hasValue {
		return value, true
	}
	if *index+1 >= len(args) {
		return "", true
	}
	*index = *index + 1
	return args[*index], true
}

func consumeRepoSandboxGoNonPathFlagValue(arg string, args []string, index *int) {
	name, _, hasValue := splitRepoSandboxGoFlagValue(arg)
	if hasValue {
		return
	}
	if !isRepoSandboxGoNonPathValueFlag(name) || *index+1 >= len(args) {
		return
	}
	*index = *index + 1
}

func splitRepoSandboxGoFlagValue(arg string) (name, value string, hasValue bool) {
	if idx := strings.IndexByte(arg, '='); idx >= 0 {
		return arg[:idx], arg[idx+1:], true
	}
	return arg, "", false
}

func isRepoSandboxGoPathFlag(name string) bool {
	switch name {
	case "-C", "-o", "-outputdir", "-coverprofile", "-cpuprofile", "-memprofile", "-mutexprofile", "-blockprofile", "-trace", "-modfile", "-overlay", "-pkgdir":
		return true
	default:
		return false
	}
}

func isRepoSandboxGoNonPathValueFlag(name string) bool {
	switch name {
	case "-p", "-tags", "-run", "-bench", "-timeout", "-count":
		return true
	default:
		return false
	}
}

func isLocalGoPackagePathArg(arg string) bool {
	if arg == "." || arg == ".." {
		return true
	}
	if filepath.IsAbs(arg) {
		return true
	}
	return strings.HasPrefix(arg, "."+"/") ||
		strings.HasPrefix(arg, ".."+"/") ||
		strings.HasPrefix(arg, "."+string(filepath.Separator)) ||
		strings.HasPrefix(arg, ".."+string(filepath.Separator))
}
