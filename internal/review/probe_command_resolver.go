package review

import (
	"path/filepath"
	"strings"
)

type probeCommandResolutionContext struct {
	RepoRoot   string
	ScratchDir string
	WorkDir    string
	Env        []string
}

func resolveProbeCommandPath(command string, ctx probeCommandResolutionContext) (string, error) {
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
	resolveBaseDir, err := resolveProbeCommandResolutionBaseDir(ctx.WorkDir)
	if err != nil {
		return "", err
	}

	candidates := buildResolverCommandCandidates(trimmedCommand, envValue(ctx.Env, "PATHEXT"))
	for _, dir := range filepath.SplitList(pathValue) {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		searchDir, err := resolveProbeCommandSearchDir(resolveBaseDir, dir)
		if err != nil {
			return "", err
		}
		for _, candidateName := range candidates {
			candidatePath := filepath.Join(searchDir, candidateName)
			resolvedPath, ok := resolveExecutablePath(candidatePath, envValue(ctx.Env, "PATHEXT"))
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

func resolveProbeCommandResolutionBaseDir(workDir string) (string, error) {
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

func resolveProbeCommandSearchDir(baseDir, pathEntry string) (string, error) {
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
