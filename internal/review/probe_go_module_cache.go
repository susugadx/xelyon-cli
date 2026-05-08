package review

import (
	"os"
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
	trimmed := strings.TrimSpace(candidate)
	if trimmed == "" || !filepath.IsAbs(trimmed) {
		return "", false
	}

	cleaned := filepath.Clean(trimmed)
	info, err := os.Lstat(cleaned)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if probePathIsInsideRoot(cleaned, repoRoot) {
		return "", false
	}

	evaluated, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		return "", false
	}
	evaluated = filepath.Clean(evaluated)
	info, err = os.Stat(evaluated)
	if err != nil || !info.IsDir() {
		return "", false
	}
	if probePathIsInsideRoot(evaluated, repoRoot) {
		return "", false
	}
	return evaluated, true
}

func probePathIsInsideRoot(pathValue, root string) bool {
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	inside, err := isPathWithinRepoRoot(filepath.Clean(root), filepath.Clean(pathValue))
	return err == nil && inside
}
