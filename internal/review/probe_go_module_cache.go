package review

import (
	"path/filepath"
	"strings"
)

func probeGoModuleCacheReadOnlyBind(baseEnv []string, repoRoot, target string) (probeProcessSandboxBind, bool) {
	source, ok := probeHostGoModuleCacheDir(baseEnv, repoRoot)
	if !ok {
		return probeProcessSandboxBind{}, false
	}
	target = filepath.Clean(target)
	if samePathForResolver(source, target) {
		return probeProcessSandboxBind{}, false
	}
	return probeProcessSandboxBind{source: source, target: target}, true
}

func probeGoModuleCacheReadOnlyBinds(baseEnv []string, repoRoot, target string) []probeProcessSandboxBind {
	bind, ok := probeGoModuleCacheReadOnlyBind(baseEnv, repoRoot, target)
	if !ok {
		return nil
	}
	return []probeProcessSandboxBind{bind}
}

func probeHostGoModuleCacheDir(baseEnv []string, repoRoot string) (string, bool) {
	for _, candidate := range probeGoModuleCacheCandidates(baseEnv) {
		resolved, ok := validateProbeHostGoModuleCacheDir(candidate, repoRoot)
		if ok {
			return resolved, true
		}
	}
	return "", false
}

func probeGoModuleCacheCandidates(baseEnv []string) []string {
	if explicit := strings.TrimSpace(envValue(baseEnv, "GOMODCACHE")); explicit != "" {
		return []string{explicit}
	}

	candidates := make([]string, 0, 2)
	if gopath := strings.TrimSpace(envValue(baseEnv, "GOPATH")); gopath != "" {
		for _, entry := range filepath.SplitList(gopath) {
			entry = strings.TrimSpace(entry)
			if entry == "" {
				continue
			}
			candidates = append(candidates, filepath.Join(entry, "pkg", "mod"))
			break
		}
		return candidates
	}

	for _, homeKey := range []string{"HOME", "USERPROFILE"} {
		home := strings.TrimSpace(envValue(baseEnv, homeKey))
		if home == "" {
			continue
		}
		candidates = append(candidates, filepath.Join(home, "go", "pkg", "mod"))
		break
	}
	return candidates
}

func validateProbeHostGoModuleCacheDir(candidate, repoRoot string) (string, bool) {
	return validateProbeHostReadOnlyDir(candidate, repoRoot)
}

func probePathIsInsideRoot(pathValue, root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	inside, err := isPathWithinRepoRoot(filepath.Clean(root), filepath.Clean(pathValue))
	return err == nil && inside
}
