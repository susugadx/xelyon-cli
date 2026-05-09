package review

const probeGoRootEnvKey = "GOROOT"

func probeHostGoRootDir(baseEnv []string, repoRoot string) (string, bool) {
	return validateProbeHostReadOnlyDir(envValue(baseEnv, probeGoRootEnvKey), repoRoot)
}

func probeHostGoRootEnv(baseEnv []string, repoRoot string) []string {
	goRoot, ok := probeHostGoRootDir(baseEnv, repoRoot)
	if !ok {
		return nil
	}
	return []string{probeGoRootEnvKey + "=" + goRoot}
}

func probeGoRootReadOnlyBind(env []string) (probeProcessSandboxBind, bool) {
	goRoot, ok := validateProbeHostReadOnlyDir(envValue(env, probeGoRootEnvKey), "")
	if !ok {
		return probeProcessSandboxBind{}, false
	}
	return probeProcessSandboxBind{source: goRoot, target: goRoot}, true
}
