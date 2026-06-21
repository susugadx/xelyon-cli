package probe

func buildScratchOnlyEnv(baseEnv []string, repoRoot string, dirs scratchOnlyDirs) []string {
	return buildIsolatedProbeEnv(baseEnv, isolatedProbeRuntimeDirs{
		RootDir:        dirs.ScratchDir,
		WorkDir:        dirs.ScratchDir,
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
		repoRootEnvKey:   scratchEnvRepoRoot,
		repoRootEnvValue: repoRoot,
		modeRootEnvKey:   scratchEnvScratchDir,
		modeRootEnvValue: dirs.ScratchDir,
	})
}
