package probe

import "strings"

var isolatedProbeInheritedEnvKeys = []string{
	"PATH",
	"PATHEXT",
	"SystemRoot",
	"WINDIR",
	"COMSPEC",
	"LANG",
	"LC_ALL",
	"LC_CTYPE",
}

type isolatedProbeEnvSpec struct {
	repoRootEnvKey   string
	repoRootEnvValue string
	modeRootEnvKey   string
	modeRootEnvValue string
}

func buildIsolatedProbeEnv(baseEnv []string, dirs isolatedProbeRuntimeDirs, spec isolatedProbeEnvSpec) []string {
	env := make([]string, 0, len(isolatedProbeInheritedEnvKeys)+24)

	env = append(env, collectIsolatedProbeInheritedEnv(baseEnv)...)
	env = append(env, probeHostGoRootEnv(baseEnv, spec.repoRootEnvValue)...)

	if spec.repoRootEnvKey != "" {
		env = append(env, spec.repoRootEnvKey+"="+spec.repoRootEnvValue)
	}
	if spec.modeRootEnvKey != "" {
		env = append(env, spec.modeRootEnvKey+"="+spec.modeRootEnvValue)
	}

	env = append(env,
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
		"GOENV=off",
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
		"GOSUMDB=off",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+isolatedProbeGitConfigGlobalFile(dirs),
		"GIT_CONFIG_SYSTEM="+isolatedProbeGitConfigSystemFile(dirs),
		"GIT_OPTIONAL_LOCKS=0",
		"GIT_TERMINAL_PROMPT=0",
		"NPM_CONFIG_USERCONFIG="+isolatedProbeNPMUserConfigFile(dirs),
		"NPM_CONFIG_GLOBALCONFIG="+isolatedProbeNPMGlobalConfigFile(dirs),
		"NPM_CONFIG_CACHE="+isolatedProbeNPMCacheDir(dirs),
		"NPM_CONFIG_PREFIX="+isolatedProbeNPMPrefixDir(dirs),
		"NPM_CONFIG_AUDIT=false",
		"NPM_CONFIG_FUND=false",
		"CARGO_HOME="+isolatedProbeCargoHomeDir(dirs),
		"CARGO_TARGET_DIR="+isolatedProbeCargoTargetDir(dirs),
		"PYTHONDONTWRITEBYTECODE=1",
		"PYTHONPYCACHEPREFIX="+dirs.PythonCacheDir,
		"PYTHONNOUSERSITE=1",
		"PIP_DISABLE_PIP_VERSION_CHECK=1",
		"PIP_NO_INDEX=1",
	)
	return env
}

func collectIsolatedProbeInheritedEnv(baseEnv []string) []string {
	base := collectEnvMap(baseEnv)
	inherited := make([]string, 0, len(isolatedProbeInheritedEnvKeys))
	seen := make(map[string]struct{}, len(isolatedProbeInheritedEnvKeys)+4)

	for _, key := range isolatedProbeInheritedEnvKeys {
		if value, ok := base[key]; ok {
			inherited = append(inherited, key+"="+value)
			seen[key] = struct{}{}
		}
	}

	for _, entry := range baseEnv {
		key, value, ok := splitEnvEntry(entry)
		if !ok || !strings.HasPrefix(key, "LC_") {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		inherited = append(inherited, key+"="+value)
		seen[key] = struct{}{}
	}

	return inherited
}

func collectEnvMap(baseEnv []string) map[string]string {
	m := make(map[string]string, len(baseEnv))
	for _, entry := range baseEnv {
		key, value, ok := splitEnvEntry(entry)
		if !ok {
			continue
		}
		m[key] = value
	}
	return m
}

func splitEnvEntry(entry string) (key, value string, ok bool) {
	idx := strings.IndexByte(entry, '=')
	if idx <= 0 {
		return "", "", false
	}
	return entry[:idx], entry[idx+1:], true
}
