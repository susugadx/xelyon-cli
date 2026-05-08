package review

const (
	hostReadOnlyEnvRepoRoot    = "XELYON_REVIEW_REPO_ROOT"
	hostReadOnlyEnvRuntimeRoot = "XELYON_REVIEW_HOST_RUNTIME_ROOT"
)

func prepareHostReadOnlyRuntimeDirs(runtimeRoot string) (isolatedProbeRuntimeDirs, error) {
	return prepareIsolatedProbeRuntimeDirs(isolatedProbeRuntimeLayout{
		rootDir:         runtimeRoot,
		workDir:         runtimeRoot,
		runtimeBaseDir:  runtimeRoot,
		createErrorName: "host_readonly runtime",
	})
}

func buildHostReadOnlyEnv(baseEnv []string, repoRoot string, dirs isolatedProbeRuntimeDirs) []string {
	return buildIsolatedProbeEnv(baseEnv, dirs, isolatedProbeEnvSpec{
		repoRootEnvKey:   hostReadOnlyEnvRepoRoot,
		repoRootEnvValue: repoRoot,
		modeRootEnvKey:   hostReadOnlyEnvRuntimeRoot,
		modeRootEnvValue: dirs.RootDir,
	})
}
