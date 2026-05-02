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
	base := collectEnvMap(baseEnv)
	env := make([]string, 0, len(scratchOnlyInheritedEnvKeys)+24)

	for _, key := range scratchOnlyInheritedEnvKeys {
		if value, ok := base[key]; ok {
			env = append(env, key+"="+value)
		}
	}

	env = append(env,
		scratchEnvRepoRoot+"="+repoRoot,
		scratchEnvScratchDir+"="+dirs.ScratchDir,
		"HOME="+dirs.HomeDir,
		"USERPROFILE="+dirs.HomeDir,
		"TMPDIR="+dirs.TempDir,
		"TEMP="+dirs.TempDir,
		"TMP="+dirs.TempDir,
		"XDG_CACHE_HOME="+dirs.XDGCacheDir,
		"XDG_CONFIG_HOME="+dirs.XDGConfigDir,
		"XDG_DATA_HOME="+dirs.XDGDataDir,
		"GOCACHE="+dirs.GoCacheDir,
		"GOMODCACHE="+dirs.GoModCacheDir,
		"GOTMPDIR="+dirs.GoTempDir,
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONPYCACHEPREFIX="+dirs.PythonCacheDir,
		"PYTHONNOUSERSITE=1",
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"PIP_NO_INDEX=1",
	)
	return env
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
