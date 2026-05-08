package review

import "strings"

var scratchOnlyInheritedEnvKeys = []string{
	"PATH",
	"PATHEXT",
	"SystemRoot",
	"WINDIR",
	"COMSPEC",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
}

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

func collectEnvMap(baseEnv []string) map[string]string {
	m := make(map[string]string, len(baseEnv))
	for _, entry := range baseEnv {
		idx := strings.IndexByte(entry, '=')
		if idx <= 0 {
			continue
		}
		key := entry[:idx]
		value := entry[idx+1:]
		m[key] = value
	}
	return m
}
