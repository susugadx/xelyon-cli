package review

const (
	repoSandboxEnvRepoRoot    = "XELYON_REVIEW_REPO_ROOT"
	repoSandboxEnvSandboxRoot = "XELYON_REVIEW_SANDBOX_ROOT"
)

func buildRepoSandboxEnv(baseEnv []string, dirs repoSandboxDirs) []string {
	return buildIsolatedProbeEnv(baseEnv, isolatedProbeRuntimeDirs{
		RootDir:        dirs.SandboxRoot,
		WorkDir:        dirs.WorktreeDir,
		HomeDir:        dirs.HomeDir,
		TempDir:        dirs.TempDir,
		CacheDir:       dirs.CacheDir,
		GoCacheDir:     dirs.GoCacheDir,
		GoModCacheDir:  dirs.GoModCacheDir,
		PythonCacheDir: dirs.PythonCacheDir,
		XDGCacheDir:    dirs.XDGCacheDir,
		GoTempDir:      dirs.GoTempDir,
		XDGConfigDir:   dirs.XDGConfigDir,
		XDGDataDir:     dirs.XDGDataDir,
	}, isolatedProbeEnvSpec{
		repoRootEnvKey:   repoSandboxEnvRepoRoot,
		repoRootEnvValue: dirs.WorktreeDir,
		modeRootEnvKey:   repoSandboxEnvSandboxRoot,
		modeRootEnvValue: dirs.SandboxRoot,
	})
}
