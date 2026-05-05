package review

import (
	"path/filepath"
	"strings"
)

type commandResolutionContext struct {
	RepoRoot   string
	ScratchDir string
	WorkDir    string
	Env        []string
}

func resolveCommandPath(command string, ctx commandResolutionContext) (string, error) {
	trimmedCommand := strings.TrimSpace(command)
	if trimmedCommand == "" {
		return "", newBlockedCommandErrorf("command is empty")
	}
	if strings.ContainsAny(trimmedCommand, `/\`) {
		return "", newBlockedCommandErrorf("command path is not allowed: %s", trimmedCommand)
	}

	pathValue := envValue(ctx.Env, "PATH")
	if strings.TrimSpace(pathValue) == "" {
		return "", newBlockedCommandErrorf("command %s could not be resolved from PATH", trimmedCommand)
	}
	resolveBaseDir, err := resolveCommandResolutionBaseDir(ctx.WorkDir)
	if err != nil {
		return "", err
	}

	pathExt := envValue(ctx.Env, "PATHEXT")
	candidates := buildResolverCommandCandidates(trimmedCommand, pathExt)
	for _, dir := range filepath.SplitList(pathValue) {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		searchDir, err := resolveCommandSearchDir(resolveBaseDir, dir)
		if err != nil {
			return "", err
		}
		for _, candidateName := range candidates {
			candidatePath := filepath.Join(searchDir, candidateName)
			resolvedPath, ok := resolveExecutablePath(candidatePath, pathExt)
			if !ok {
				continue
			}

			if err := rejectExecutableInsideBlockedRoots(resolvedPath, ctx.RepoRoot, ctx.ScratchDir); err != nil {
				return "", err
			}
			return resolvedPath, nil
		}
	}

	return "", newBlockedCommandErrorf("command %s could not be resolved from PATH", trimmedCommand)
}

func resolveCommandResolutionBaseDir(workDir string) (string, error) {
	trimmed := strings.TrimSpace(workDir)
	if trimmed == "" {
		return "", newBlockedCommandErrorf("command lookup workdir is required")
	}

	absDir, err := filepath.Abs(trimmed)
	if err != nil {
		return "", newBlockedCommandErrorf("failed to resolve command lookup base directory %q: %v", workDir, err)
	}
	return filepath.Clean(absDir), nil
}

func resolveCommandSearchDir(baseDir, pathEntry string) (string, error) {
	trimmed := strings.TrimSpace(pathEntry)
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}

	return filepath.Clean(filepath.Join(baseDir, trimmed)), nil
}

func envValue(env []string, key string) string {
	if key == "" {
		return ""
	}

	var fallback string
	for _, entry := range env {
		idx := strings.IndexByte(entry, '=')
		if idx <= 0 {
			continue
		}
		currentKey := entry[:idx]
		if currentKey == key {
			fallback = entry[idx+1:]
			continue
		}
		if strings.EqualFold(currentKey, key) {
			fallback = entry[idx+1:]
		}
	}
	return fallback
}
